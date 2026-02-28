package actions

import (
	"sync"

	"notifications-controller/internal/config"
	"notifications-controller/internal/slack"
	t "notifications-controller/internal/telemetry"
)

type Runner struct {
	stop chan struct{}
	once sync.Once
}

func (instance *Runner) Start() {
	if !config.Config.SlackSocketModeEnabled {
		t.Log.Info("Slack socket mode is disabled")
		return
	}

	if slack.Client == nil {
		t.Log.Warn("Slack client is not initialized, skipping socket mode startup")
		return
	}

	if instance.stop != nil {
		return
	}

	instance.stop = make(chan struct{})

	go func() {
		if err := slack.Client.RunSocketMode(instance.stop); err != nil {
			t.Log.Error("Slack socket mode exited", "error", err.Error())
		}
	}()
}

func (instance *Runner) Stop() {
	instance.once.Do(func() {
		if instance.stop != nil {
			close(instance.stop)
			instance.stop = nil
		}
	})
}
