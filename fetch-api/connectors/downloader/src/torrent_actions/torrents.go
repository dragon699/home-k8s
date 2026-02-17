package action_scheduler

import (
	"common/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"connector-downloader/settings"
	"connector-downloader/src/dto/response"
	"connector-downloader/src/qbittorrent"
	t "connector-downloader/src/telemetry"

	"github.com/go-co-op/gocron"
)


type ActionChecker struct {
	Scheduler *gocron.Scheduler
}

func (instance *ActionChecker) CreateSchedule() {
	t.Log.Info("Scheduling qBittorrent action checks..")
	instance.runActions()

	if settings.Config.TorrentActionsJobID == nil {
		if instance.Scheduler == nil {
			return
		}

		nextCheckTime := instance.getNextCheckTime()
		settings.Config.TorrentActionsNextCheck = &nextCheckTime

		jobTag := "torrent_actions"
		job, _ := instance.Scheduler.Every(settings.Config.TorrentActionsIntervalSeconds).Seconds().Do(instance.runActions)
		job.Tag(jobTag)
		settings.Config.TorrentActionsJobID = &jobTag
	}
}

func (instance *ActionChecker) getNextCheckTime() string {
	ts := time.Now().Add(
		time.Duration(
			settings.Config.TorrentActionsIntervalSeconds,
		) * time.Second,
	)

	return ts.Format("2006-01-02T15:04:05")
}

func (instance *ActionChecker) runActions() {
	lastCheckTime := time.Now().Format("2006-01-02T15:04:05")
	settings.Config.TorrentActionsLastCheck = &lastCheckTime

	t.Log.Debug("Running action checks")

	torrents, err := instance.getTorrents()

	if err != nil {
		fmt.Println()
		t.Log.Error("Failed to fetch torrents list from qBittorrent", "error", err.Error())
		nextCheckTime := instance.getNextCheckTime()
		settings.Config.TorrentActionsNextCheck = &nextCheckTime

		return
	}

	for _, torrent := range torrents.Items {
		if (torrent.Meta.ManagedBy != "connector-downloader") || (torrent.ProgressPercentage < 100) {
			t.Log.Debug(
				"Skipping torrent",
				"name", torrent.Name,
				"hash", torrent.Hash,
				"progress_percentage", torrent.ProgressPercentage,
			)
			continue
		}

		t.Log.Debug("Checking torrent", "name", torrent.Name, "hash", torrent.Hash)

		for _, action := range torrent.Meta.ScheduledActions {
			if ! (action.Status == "pending") {
				continue
			}

			qbittorrent.Client.StopTorrent(torrent.Hash)

			if action.Category == "jellyfin" {
				switch action.Name {
				case "rename":
					torrentContent, err := qbittorrent.Client.GetTorrentContent(torrent.Hash)
					
					if err != nil {
						t.Log.Error("Failed to get torrent content", "error", err.Error())
						continue
					}

					var renameFailed bool = false

					for _, file := range torrentContent {
						if file["progress"].(float64) < 1 {
							continue
						}

						filePath := path.Dir(file["name"].(string))
						fileName := path.Base(file["name"].(string))
						fileExt := path.Ext(fileName)
						fileNameNew := fmt.Sprintf(
							"%s%s",
							utils.BeautifyMovieName(fileName),
							fileExt,
						)

						srcFile := path.Join(torrent.SavePath, file["name"].(string))
						destPath := path.Join(torrent.SavePath, filePath)
						destFile := path.Join(destPath, fileNameNew)

						fmt.Printf("src: %s\ndest: %s\n", srcFile, destFile)

						err := os.Rename(srcFile, destFile)

						if err != nil {
							t.Log.Error("Failed to rename file", "error", err.Error())
							renameFailed = true

							continue
						}
					}

					qbittorrent.Client.RemoveTorrentTags(
						torrent.Hash,
						[]string{
							"jellyfin:pending=rename",
						},
					)

					if renameFailed {
						qbittorrent.Client.AddTorrentTags(
							torrent.Hash,
							[]string{
								"jellyfin:failed=rename",
							},
						)

						continue
					}

					qbittorrent.Client.AddTorrentTags(
						torrent.Hash,
						[]string{
							"jellyfin:completed=rename",
						},
					)
				}
			}
		}
	}
}

func (instance *ActionChecker) getTorrents() (*response.BaseResponse[response.Torrent], error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/torrents", settings.Config.ListenUrl),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch response: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var torrents response.BaseResponse[response.Torrent]
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &torrents, nil
}
