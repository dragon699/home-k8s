package actions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
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
					op := runner.JellyfinRename(torrent, torrentContentFiles, torrentContentNewFileNames)

					if op != nil {
						continue
					}

				case "find_subs":
					var subsDownloadedCount int = 0
					var subsAlreadyPresentCount int = 0

					jellyfin.Client.RefreshLibrary()
					time.Sleep(2 * time.Second)

					jellyfinItems, err := jellyfin.Client.GetItems()
					if err != nil {
						t.Log.Error("Failed to get Jellyfin items", "error", err.Error())
						qbittorrent.Client.DeleteTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=pending"})
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=failed"})

						continue
					}

					for _, item := range jellyfinItems {
						if hasSubtitles, ok := item["HasSubtitles"].(bool); ok && hasSubtitles {
							var subtitlesFound bool = false

							if mediaStreams, ok := item["MediaStreams"].([]map[string]any); ok {
								for _, stream := range mediaStreams {
									if stream["Type"] == "Subtitle" {
										if subtitleLanguage, ok := stream["Language"].(string); ok && subtitleLanguage == config.Config.JellyfinSubtitlesDefaultLanguage[:3] {
											subtitlesFound = true
											break
										}
									}
								}
							}

							if subtitlesFound {
								subsAlreadyPresentCount += 1
								continue
							}
						}

						itemFile := filepath.Base(item["Path"].(string))

						if slices.Contains(torrentContentNewFileNames, itemFile) {
							err = instance.downloadSubtitlesInJellyfin(item["Id"].(string), config.Config.JellyfinSubtitlesDefaultLanguage)
							if err != nil {
								t.Log.Error("Failed to download subtitles in Jellyfin", "error", err.Error())
								continue
							}

							subsDownloadedCount += 1
						}
					}

					qbittorrent.Client.DeleteTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=pending"})

					if subsAlreadyPresentCount == len(torrentContentNewFileNames) {
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=already_present"})
					} else if (subsDownloadedCount + subsAlreadyPresentCount) == len(torrentContentNewFileNames) {
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=completed"})
					} else {
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=partially_completed"})
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

func (instance *ActionsRunner) downloadSubtitlesInJellyfin(itemID string, language string) error {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/Items/%s/RemoteSearch/Subtitles/%s", config.Config.JellyfinUrl, itemID, language),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Emby-Token", config.Config.JellyfinAPIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch response: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result []map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result) == 0 {
		return fmt.Errorf("no subtitles found in Jellyfin for item ID: %s and language: %s", itemID, language)
	}

	jellyfinSubtitlesID := result[0]["Id"].(string)

	req, err = http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/Items/%s/RemoteSearch/Subtitles/%s", config.Config.JellyfinUrl, itemID, jellyfinSubtitlesID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Emby-Token", config.Config.JellyfinAPIKey)

	resp, err = httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch response: %w", err)
	}

	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
