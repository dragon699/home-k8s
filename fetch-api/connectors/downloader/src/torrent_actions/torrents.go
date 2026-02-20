package torrent_actions

import (
	"common/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"connector-downloader/settings"
	"connector-downloader/src/dto/response"
	"connector-downloader/src/notifications"
	"connector-downloader/src/qbittorrent"
	t "connector-downloader/src/telemetry"

	"github.com/go-co-op/gocron"
)

type ActionsRunner struct {
	Scheduler *gocron.Scheduler
}

func (instance *ActionsRunner) CreateSchedule() {
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

func (instance *ActionsRunner) getNextCheckTime() string {
	ts := time.Now().Add(
		time.Duration(
			settings.Config.TorrentActionsIntervalSeconds,
		) * time.Second,
	)

	return ts.Format("2006-01-02T15:04:05")
}

func (instance *ActionsRunner) runActions() {
	lastCheckTime := time.Now().Format("2006-01-02T15:04:05")
	settings.Config.TorrentActionsLastCheck = &lastCheckTime

	torrents, err := instance.getTorrents()

	if err != nil {
		fmt.Println()
		t.Log.Error("Failed to fetch torrents list from qBittorrent", "error", err.Error())
		nextCheckTime := instance.getNextCheckTime()
		settings.Config.TorrentActionsNextCheck = &nextCheckTime

		return
	}

	for _, torrent := range torrents.Items {
		if torrent.ProgressPercentage < 100 {
			for _, action := range torrent.Meta.ScheduledActions {
				if (action.Category == "slack") && (action.Name == "notify") && (action.Status == "pending") {
					vars := notifications.NotificationTorrentsVars{
						TorrentName:    torrent.Name,
						Category:       torrent.Category,
						QBittorrentURL: settings.Config.QBittorrentPublicUrl,
						JellyfinURL:    settings.Config.JellyfinUrl,
					}

					err = notifications.SendSlackNotification(
						settings.Config.SlackNotificationsWebhookUrl,
						"templates/torrents/slack_initial.json",
						vars,
					)
					if err != nil {
						t.Log.Error("Failed to send slack notification for a torrent!", "error", err.Error())

						qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"slack:notify=pending"})
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"slack:notify=failed"})

						break
					}

					qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"slack:notify=pending"})
					qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"slack:notify=initial"})

					break
				}
			}

			continue
		} else {
			for _, action := range torrent.Meta.ScheduledActions {
				if (action.Category == "slack") && (action.Name == "notify") && (action.Status == "initial") {
					vars := notifications.NotificationTorrentsVars{
						TorrentName:    torrent.Name,
						Category:       torrent.Category,
						QBittorrentURL: settings.Config.QBittorrentPublicUrl,
						JellyfinURL:    settings.Config.JellyfinUrl,
					}

					err = notifications.SendSlackNotification(
						settings.Config.SlackNotificationsWebhookUrl,
						"templates/torrents/slack_completed.json",
						vars,
					)
					if err != nil {
						t.Log.Error("Failed to send slack notification for a completed torrent!", "error", err.Error())

						qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"slack:notify=initial"})
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"slack:notify=failed"})

						break
					}

					qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"slack:notify=initial"})
					qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"slack:notify=completed"})

					break
				}
			}
		}

		if torrent.Meta.ManagedBy != "connector-downloader" {
			continue
		}

		t.Log.Debug("Running tag actions against completed torrent", "name", torrent.Name, "hash", torrent.Hash)

		var hasPendingActions bool = false

		for _, action := range torrent.Meta.ScheduledActions {
			if !(action.Status == "pending") {
				continue
			}

			hasPendingActions = true
			qbittorrent.Client.StopTorrent(torrent.Hash)

			if action.Category == "jellyfin" {
				torrentContent, err := qbittorrent.Client.GetTorrentContent(torrent.Hash)

				if err != nil {
					t.Log.Error("Failed to get torrent content", "error", err.Error())
					continue
				}

				var allowedExtensions = []string{
					".mkv", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v",
					".mpg", ".mpeg", ".ts", ".m2ts", ".mts", ".3gp", ".3g2", ".ogv",
					".vob", ".asf", ".rm", ".rmvb", ".divx", ".f4v", ".mxf", ".mpv",
					".qt", ".dat", ".amv", ".y4m",
				}

				for _, file := range torrentContent {
					if file["progress"].(float64) < 1 {
						os.Remove(path.Join(torrent.SavePath, file["name"].(string)))
						continue
					}
				}

				switch action.Name {
				case "find_subs":
					var findSubsFailed bool = false

					instance.refreshJellyfinLibrary()
					time.Sleep(2 * time.Second)

					jellyfinTorrentFolders, err := instance.getJellyfinItemFolders(
						utils.BeautifyMovieName(filepath.Base(torrent.FilesPath)),
					)
					if err != nil {
						t.Log.Error("Failed to get Jellyfin items for torrent folder", "error", err.Error())
						qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=pending"})
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=failed"})

						continue
					}

					jellyfinTorrentFolderID := jellyfinTorrentFolders[0]["Id"].(string)

					jellyfinItems, err := instance.getJellyfinItems(jellyfinTorrentFolderID)
					if err != nil {
						t.Log.Error("Failed to get Jellyfin items for torrent folder", "error", err.Error())
						qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=pending"})
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=failed"})

						continue
					}

					for _, item := range jellyfinItems {
						if item["HasSubtitles"].(bool) {
							continue
						}

						err = instance.downloadSubtitlesInJellyfin(item["Id"].(string), "english")
						if err != nil {
							t.Log.Error("Failed to download subtitles in Jellyfin", "error", err.Error())
							findSubsFailed = true

							continue
						}
					}

					if findSubsFailed {
						qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=pending"})
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=failed"})
					}

					qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=pending"})
					qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=completed"})

				case "rename":
					var renameFailed bool = false

					for _, file := range torrentContent {
						filePath := path.Dir(file["name"].(string))
						fileName := path.Base(file["name"].(string))
						fileExt := path.Ext(fileName)

						if !slices.Contains(allowedExtensions, fileExt) {
							continue
						}

						fileNameBase := strings.TrimSuffix(fileName, fileExt)

						fileNameNew := fmt.Sprintf(
							"%s%s",
							utils.BeautifyMovieName(fileNameBase),
							fileExt,
						)

						srcFile := path.Join(torrent.SavePath, file["name"].(string))
						destPath := path.Join(torrent.SavePath, filePath)
						destFile := path.Join(destPath, fileNameNew)

						err := os.Rename(srcFile, destFile)
						if err != nil {
							renameFailed = true
						}
					}

					dirPath := filepath.Dir(torrent.FilesPath)
					dirName := filepath.Base(torrent.FilesPath)
					dirPathNew := path.Join(
						dirPath,
						utils.BeautifyMovieName(dirName),
					)

					err = os.Rename(torrent.FilesPath, dirPathNew)
					if err != nil {
						renameFailed = true
					}

					qbittorrent.Client.RemoveTorrentTags(torrent.Hash, []string{"jellyfin:rename=pending"})

					if renameFailed {
						qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:rename=failed"})

						continue
					}

					qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:rename=completed"})
				}
			}
		}

		if ! hasPendingActions {
			qbittorrent.Client.RemoveTorrent(torrent.Hash, false)

			if torrent.Category == "jellyfin" {
				instance.refreshJellyfinLibrary()
			}
		}
	}
}

func (instance *ActionsRunner) getTorrents() (*response.BaseResponse[response.Torrent], error) {
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

func (instance *ActionsRunner) refreshJellyfinLibrary() error {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/Library/Refresh", settings.Config.JellyfinUrl),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Token", settings.Config.JellyfinAPIKey)

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

	return nil
}

func (instance *ActionsRunner) getJellyfinItemFolders(searchText string) ([]map[string]any, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	reqParams := url.Values{}
	reqParams.Set("Recursive", "true")
	reqParams.Set("searchTerm", searchText)
	reqParams.Set("IncludeItemTypes", "Folder")

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/Items?%s", settings.Config.JellyfinUrl, reqParams.Encode()),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Emby-Token", settings.Config.JellyfinAPIKey)

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

	var result struct {
		Items []map[string]any `json:"Items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("no items found in Jellyfin with search term: %s", searchText)
	}

	return result.Items, nil
}

func (instance *ActionsRunner) getJellyfinItems(parentID string) ([]map[string]any, error) {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	reqParams := url.Values{}
	reqParams.Set("Recursive", "true")
	reqParams.Set("ParentId", parentID)
	reqParams.Set("IncludeItemTypes", "Episode,Movie,Series,Trailer,Video")

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/Items?%s", settings.Config.JellyfinUrl, reqParams.Encode()),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Emby-Token", settings.Config.JellyfinAPIKey)

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

	var result struct {
		Items []map[string]any `json:"Items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("no items found in Jellyfin")
	}

	return result.Items, nil
}

func (instance *ActionsRunner) downloadSubtitlesInJellyfin(itemID string, language string) error {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/Items/%s/RemoteSearch/Subtitles/%s", settings.Config.JellyfinUrl, itemID, language),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Emby-Token", settings.Config.JellyfinAPIKey)

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

	var result struct {
		Items []map[string]any `json:"Items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Items) == 0 {
		return fmt.Errorf("no subtitles found in Jellyfin for item ID: %s and language: %s", itemID, language)
	}

	jellyfinSubtitlesID := result.Items[0]["Id"].(string)

	req, err = http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/Items/%s/RemoteSearch/Subtitles/%s", settings.Config.JellyfinUrl, itemID, jellyfinSubtitlesID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Emby-Token", settings.Config.JellyfinAPIKey)

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
