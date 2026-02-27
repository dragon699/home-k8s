package actions

import (
	"slices"
	"time"

	torrentActions "connector-downloader/internal/actions/torrents"
	"connector-downloader/internal/config"
	"connector-downloader/internal/http/dto/response"
	"connector-downloader/internal/jellyfin"
	"connector-downloader/internal/mapper"
	"connector-downloader/internal/qbittorrent"
	t "connector-downloader/internal/telemetry"
)

func (instance *ActionsRunner) getNextCheckTime() string {
	ts := time.Now().Add(
		time.Duration(
			config.Config.TorrentActionsIntervalSeconds,
		) * time.Second,
	)

	return ts.Format("2006-01-02T15:04:05")
}

func (instance *ActionsRunner) run() {
	lastCheckTime := time.Now().Format("2006-01-02T15:04:05")
	config.Config.TorrentActionsLastCheck = &lastCheckTime
	runner := torrentActions.Actions{}

	rawTorrents, err := qbittorrent.Client.ListTorrents()

	if err != nil {
		t.Log.Error("Failed to fetch torrents list from qBittorrent", "error", err.Error())
		nextCheckTime := instance.getNextCheckTime()
		config.Config.TorrentActionsNextCheck = &nextCheckTime

		return
	}

	torrents := mapper.TorrentsFromQBittorrent(rawTorrents)

	for _, torrent := range torrents {
		if torrent.ProgressPercentage < 100 {
			for _, action := range torrent.Meta.ScheduledActions {
				if (action.Category == "slack") && (action.Name == "notify") && (action.Status == "pending") {
					vars := torrentActions.TorrentsSlackNotificationVars{
						TorrentName: torrent.Name,
						Category:    torrent.Category,
					}

					err := runner.SlackNotify(
						"torrents_initial",
						vars,
					)

					if err != nil {
						t.Log.Error("Failed to send slack notification for a torrent!", "error", err.Error())

						qbittorrent.Client.DeleteTorrentTags(torrent.Hash, []string{"slack:notify=pending"})
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"slack:notify=failed"})

						break
					}

					qbittorrent.Client.DeleteTorrentTags(torrent.Hash, []string{"slack:notify=pending"})
					qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"slack:notify=initial"})

					break
				}
			}

			continue
		} else {
			for _, action := range torrent.Meta.ScheduledActions {
				if (action.Category == "slack") && (action.Name == "notify") && (action.Status == "initial") {
					vars := torrentActions.TorrentsSlackNotificationVars{
						TorrentName: torrent.Name,
						Category:    torrent.Category,
					}

					err = runner.SlackNotify(
						"torrents_completed",
						vars,
					)

					if err != nil {
						t.Log.Error("Failed to send slack notification for a completed torrent!", "error", err.Error())

						qbittorrent.Client.DeleteTorrentTags(torrent.Hash, []string{"slack:notify=initial"})
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"slack:notify=failed"})

						break
					}

					qbittorrent.Client.DeleteTorrentTags(torrent.Hash, []string{"slack:notify=initial"})
					qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"slack:notify=completed"})

					break
				}
			}
		}

		if torrent.Meta.ManagedBy != "connector-downloader" {
			continue
		}

		var hasPendingActions bool = false
		var actionsOrder = map[string]int{
			"rename":    0,
			"find_subs": 1,
		}

		slices.SortStableFunc(torrent.Meta.ScheduledActions, func(a, b response.TorrentMetaScheduledAction) int {
			orderA, okA := actionsOrder[a.Name]
			orderB, okB := actionsOrder[b.Name]

			if !okA {
				orderA = 999
			}

			if !okB {
				orderB = 999
			}

			return orderA - orderB
		})

		for _, action := range torrent.Meta.ScheduledActions {
			if !(action.Status == "pending") {
				continue
			}

			hasPendingActions = true
			qbittorrent.Client.StopTorrent(torrent.Hash)

			if action.Category == "jellyfin" {
				torrentContentFiles, torrentContentNewFileNames, err := runner.TorrentPostDownload(torrent)

				if err != nil {
					t.Log.Error("Failed to execute torrent post-download actions", "error", err.Error())
					continue
				}

				switch action.Name {
				case "rename":
					err := runner.JellyfinRename(torrent, torrentContentFiles, torrentContentNewFileNames)

					if err != nil {
						continue
					}

				case "find_subs":
					err := runner.JellyfinFindSubs(torrent, torrentContentNewFileNames)

					if err != nil {
						continue
					}
				}
			}
		}

		if !hasPendingActions {
			if torrent.Category == "jellyfin" {
				jellyfin.Client.RefreshLibrary()
			}
		}
	}
}
