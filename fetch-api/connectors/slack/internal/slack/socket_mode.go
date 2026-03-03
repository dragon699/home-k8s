package slack

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"connector-slack/internal/config"
	t "connector-slack/internal/telemetry"

	slackapi "github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

var SocketMode *SlackSocketMode

type SlackSocketMode struct {
	Debug  bool
	Socket *socketmode.Client
	stop   chan struct{}
	once   sync.Once
}

func (instance *SlackSocketMode) Init() {
	instance.Debug = config.Config.SlackSocketDebug
	instance.Socket = socketmode.New(Client.Client)
}

func (instance *SlackSocketMode) Start() error {
	if instance.stop != nil {
		return nil
	}

	instance.stop = make(chan struct{})

	go func() {
		if err := instance.Run(instance.stop); err != nil {
			t.Log.Error("Slack socket mode exited", "error", err.Error())
		}
	}()

	return nil
}

func (instance *SlackSocketMode) Stop() {
	instance.once.Do(func() {
		if instance.stop != nil {
			close(instance.stop)
			instance.stop = nil
		}
	})
}

func (instance *SlackSocketMode) Run(stop <-chan struct{}) error {
	ctx, cancel := context.WithCancel(context.Background())
	handler := socketmode.NewSocketmodeHandler(instance.Socket)

	handler.Handle(socketmode.EventTypeConnecting, func(evt *socketmode.Event, client *socketmode.Client) {
		t.Log.Info("Slack socket mode connecting")
	})

	handler.Handle(socketmode.EventTypeConnectionError, func(evt *socketmode.Event, client *socketmode.Client) {
		t.Log.Error("Slack socket mode connection error")
	})

	handler.Handle(socketmode.EventTypeConnected, func(evt *socketmode.Event, client *socketmode.Client) {
		t.Log.Info("Slack socket mode connected")
	})

	handler.HandleInteraction(slackapi.InteractionTypeBlockActions, func(evt *socketmode.Event, client *socketmode.Client) {
		callback, ok := evt.Data.(slackapi.InteractionCallback)
		if !ok {
			t.Log.Warn("Slack interactive payload had unexpected type")
			return
		}

		if evt.Request != nil {
			client.Ack(*evt.Request)
		}

		instance.handleInteraction(callback)
	})

	go func() {
		<-stop
		cancel()
	}()

	if err := handler.RunEventLoopContext(ctx); err != nil && ctx.Err() == nil {
		return err
	}

	return nil
}

func (instance *SlackSocketMode) handleInteraction(callback slackapi.InteractionCallback) {
	if len(callback.ActionCallback.BlockActions) == 0 {
		return
	}

	action := callback.ActionCallback.BlockActions[0]
	message := fmt.Sprintf("Action received: %s", action.ActionID)

	if strings.TrimSpace(action.Value) != "" {
		message = fmt.Sprintf("%s (%s)", message, action.Value)
	}

	_, err := Client.Client.PostEphemeral(
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
