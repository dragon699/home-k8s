package slack

import (
	"context"
	"encoding/json"
	"fmt"

	"connector-slack/internal/config"

	slackapi "github.com/slack-go/slack"
)

var Client *SlackClient

type SlackClient struct {
	BotToken          string
	AppToken          string
	DefaultChannel    string
	APIBaseURL        string
	APIDefaultHeaders map[string]string
	Client            *slackapi.Client
}

func (instance *SlackClient) Init() {
	instance.BotToken = config.Config.SlackBotToken
	instance.AppToken = config.Config.SlackAppToken
	instance.DefaultChannel = config.Config.SlackDefaultChannelID

	instance.APIBaseURL = "https://slack.com/api"
	instance.APIDefaultHeaders = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", instance.BotToken),
		"Content-Type":  "application/json; charset=utf-8",
	}

	instance.Client = slackapi.New(
		instance.BotToken,
		slackapi.OptionAppLevelToken(instance.AppToken),
		slackapi.OptionDebug(config.Config.SlackSocketDebug),
	)
}

func (instance *SlackClient) Ping() error {
	_, err := instance.Client.AuthTestContext(context.Background())

	if err != nil {
		return config.NewConnectionError("Slack auth test failed", err)
	}

	return nil
}

func (instance *SlackClient) SendMsg(channelID string, blocks []map[string]any, attachments []map[string]any, options ...slackapi.MsgOption) (*Response, error) {
	opts := append(
		[]slackapi.MsgOption{
			slackapi.MsgOptionBlocks(toBlockSet(blocks)...),
			slackapi.MsgOptionAttachments(toAttachmentSet(attachments)...),
		},
		options...,
	)
	ch, ts, err := instance.Client.PostMessage(channelID, opts...)

	if err != nil {
		return nil, config.NewUpstreamError(
			fmt.Sprintf("chat.postMessage failed for channel %q", channelID),
			0,
			nil,
			err,
		)
	}

	return &Response{
		Channel:   ch,
		Timestamp: ts,
	}, nil
}

func (instance *SlackClient) SendMsgFromTemplate(templateName string, templateVars any) (*Response, error) {
	raw, err := RenderTemplate(templateName, templateVars)
	if err != nil {
		return nil, fmt.Errorf("failed to render notification template %q: %w", templateName, err)
	}

	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal notification template %q: %w", templateName, err)
	}

	var msg Message
	if err := json.Unmarshal(rawJSON, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notification template %q: %w", templateName, err)
	}

	opts := []slackapi.MsgOption{
		slackapi.MsgOptionText(msg.Text, false),
		slackapi.MsgOptionBlocks(toBlockSet(msg.Blocks)...),
		slackapi.MsgOptionAttachments(toAttachmentSet(msg.Attachments)...),
	}

	if msg.Username != "" {
		opts = append(opts, slackapi.MsgOptionUsername(msg.Username))
	}

	if msg.IconURL != "" {
		opts = append(opts, slackapi.MsgOptionIconURL(msg.IconURL))
	}

	ch, ts, err := instance.Client.PostMessage(msg.Channel, opts...)
	if err != nil {
		return nil, config.NewUpstreamError(
			fmt.Sprintf("chat.postMessage failed for %q", templateName),
			0,
			nil,
			err,
		)
	}

	return &Response{
		Channel:   ch,
		Timestamp: ts,
	}, nil
}
