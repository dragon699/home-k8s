package slack

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"common/utils"
	"connector-notifications/internal/config"
	t "connector-notifications/internal/telemetry"

	slackapi "github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

var Client *SlackClient

type SlackClient struct {
	BotToken          string
	AppToken          string
	DefaultChannel    string
	GrafanaChannel    string
	DownloaderChannel string
	SocketModeEnabled bool
	SocketDebug       bool
	PostMessageURL    string
	DefaultHeaders    map[string]string
	HTTPClient        *http.Client
	API               *slackapi.Client
	Socket            *socketmode.Client
}

func (instance *SlackClient) Init() {
	instance.BotToken = config.Config.SlackBotToken
	instance.AppToken = config.Config.SlackAppToken
	instance.DefaultChannel = config.Config.SlackDefaultChannel
	instance.GrafanaChannel = config.Config.SlackGrafanaChannel
	instance.DownloaderChannel = config.Config.SlackDownloaderChannel
	instance.SocketModeEnabled = config.Config.SlackSocketModeEnabled
	instance.SocketDebug = config.Config.SlackSocketDebug
	instance.PostMessageURL = "https://slack.com/api/chat.postMessage"
	instance.DefaultHeaders = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", instance.BotToken),
		"Content-Type":  "application/json; charset=utf-8",
	}
	instance.HTTPClient = &http.Client{
		Timeout: 15 * time.Second,
	}

	if instance.BotToken != "" {
		instance.API = slackapi.New(
			instance.BotToken,
			slackapi.OptionHTTPClient(instance.HTTPClient),
			slackapi.OptionAppLevelToken(instance.AppToken),
			slackapi.OptionDebug(instance.SocketDebug),
		)
	}

	if instance.SocketModeEnabled && instance.API != nil && instance.AppToken != "" {
		instance.Socket = socketmode.New(instance.API)
	}
}

func (instance *SlackClient) Ping() error {
	if instance.API == nil {
		return config.NewConnectionError("Slack bot token is not configured", nil)
	}

	_, err := instance.API.AuthTestContext(context.Background())
	if err != nil {
		return config.NewConnectionError("Slack auth test failed", err)
	}

	return nil
}

func (instance *SlackClient) SendTemplateMessage(templateName string, templateVars any) (*SendResult, error) {
	if instance.BotToken == "" {
		return nil, config.NewConnectionError("Slack bot token is not configured", nil)
	}

	body, err := RenderTemplate(templateName, templateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to render Slack template %q: %w", templateName, err)
	}

	channel := resolveChannel(templateName, body, instance)
	if channel == "" {
		return nil, fmt.Errorf("no Slack channel configured for template %q", templateName)
	}

	body["channel"] = channel

	req := &utils.Req{Client: instance.HTTPClient}
	resp, err := req.POST(instance.PostMessageURL, instance.DefaultHeaders, nil, body)
	if err != nil {
		return nil, wrapReqError("chat.postMessage failed", err)
	}

	payload, ok := resp.Body.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Slack returned an unexpected response shape")
	}

	if okValue, ok := payload["ok"].(bool); !ok || !okValue {
		return nil, config.NewUpstreamError(
			fmt.Sprintf("Slack API rejected %q notification", templateName),
			resp.StatusCode,
			mustJSON(payload),
			nil,
		)
	}

	return &SendResult{
		Channel:   asString(payload["channel"]),
		Timestamp: asString(payload["ts"]),
	}, nil
}

func (instance *SlackClient) RunSocketMode(stop <-chan struct{}) error {
	if !instance.SocketModeEnabled {
		return nil
	}

	if instance.Socket == nil {
		return fmt.Errorf("socket mode is enabled but Slack app token is not configured")
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-stop
		cancel()
	}()

	go func() {
		if err := instance.Socket.RunContext(ctx); err != nil {
			t.Log.Error("Slack socket mode runtime failed", "error", err.Error())
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil

		case evt, ok := <-instance.Socket.Events:
			if !ok {
				return nil
			}

			switch evt.Type {
			case socketmode.EventTypeConnecting:
				t.Log.Info("Slack socket mode connecting")

			case socketmode.EventTypeConnectionError:
				t.Log.Error("Slack socket mode connection error")

			case socketmode.EventTypeConnected:
				t.Log.Info("Slack socket mode connected")

			case socketmode.EventTypeInteractive:
				callback, ok := evt.Data.(slackapi.InteractionCallback)
				if !ok {
					t.Log.Warn("Slack interactive payload had unexpected type")
					continue
				}

				if evt.Request != nil {
					instance.Socket.Ack(*evt.Request)
				}

				instance.handleInteraction(callback)
			}
		}
	}
}

func (instance *SlackClient) handleInteraction(callback slackapi.InteractionCallback) {
	if len(callback.ActionCallback.BlockActions) == 0 {
		return
	}

	action := callback.ActionCallback.BlockActions[0]
	message := fmt.Sprintf("Action received: %s", action.ActionID)
	if strings.TrimSpace(action.Value) != "" {
		message = fmt.Sprintf("%s (%s)", message, action.Value)
	}

	_, err := instance.API.PostEphemeral(
		callback.Channel.ID,
		callback.User.ID,
		slackapi.MsgOptionText(message, false),
	)
	if err != nil {
		t.Log.Error("Failed to post Slack ephemeral action response", "error", err.Error(), "action_id", action.ActionID)
		return
	}

	t.Log.Info("Handled Slack interactive action", "action_id", action.ActionID, "user", callback.User.Name)
}
