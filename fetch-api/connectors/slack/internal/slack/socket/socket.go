package socket

import (
	"context"
	"encoding/json"
	"sync"

	"connector-slack/internal/config"
	"connector-slack/internal/slack"
	"connector-slack/internal/slack/socket/handlers/grafana"
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
	instance.Socket = socketmode.New(slack.Client.Client)
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

	d, _ := json.Marshal(callback)
	t.Log.Info("Interaction payload received", "payload", string(d))

	action := callback.ActionCallback.BlockActions[0]

	switch action.ActionID {
	case "grafana_alert_button_investigate":
		grafana.ButtonInvestigate(action.Value)
	}

	t.Log.Info("Handled Slack interactive action", "action_id", action.ActionID, "user", callback.User.Name)
}
