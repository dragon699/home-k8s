package actions

import (
	"connector-downloader/internal/config"
	t "connector-downloader/internal/telemetry"

	"github.com/go-co-op/gocron"

)

type ActionsRunner struct {
	Scheduler *gocron.Scheduler
}

func (instance *ActionsRunner) CreateSchedule() {
	t.Log.Info("Scheduling qBittorrent checks..")
	instance.runActions()

	if config.Config.TorrentActionsJobID == nil {
		if instance.Scheduler == nil {
			return
		}

		nextCheckTime := instance.getNextCheckTime()
		config.Config.TorrentActionsNextCheck = &nextCheckTime

		jobTag := "torrent_actions"
		job, _ := instance.Scheduler.Every(config.Config.TorrentActionsIntervalSeconds).Seconds().Do(instance.runActions)
		job.Tag(jobTag)
		config.Config.TorrentActionsJobID = &jobTag
	}
}
