package torrents

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"common/utils"
	"connector-downloader/internal/config"
	"connector-downloader/internal/http/dto/response"
	"connector-downloader/internal/jellyfin"
	"connector-downloader/internal/qbittorrent"
	"connector-downloader/internal/slack"
	t "connector-downloader/internal/telemetry"
)

type Actions struct{}

func (instance *Actions) TorrentPostDownload(torrent response.Torrent) ([]map[string]any, []string, error) {
	torrentContent, err := qbittorrent.Client.GetTorrentContent(torrent.Hash)

	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get torrent content: %w", err)
	}

	var torrentContentFiles = []map[string]any{}
	var torrentContentNewFileNames = []string{}
	var allowedExtensions = []string{
		".mkv", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v",
		".mpg", ".mpeg", ".ts", ".m2ts", ".mts", ".3gp", ".3g2", ".ogv",
		".vob", ".asf", ".rm", ".rmvb", ".divx", ".f4v", ".mxf", ".mpv",
		".qt", ".dat", ".amv", ".y4m",
	}

	for _, file := range torrentContent {
		if file["progress"].(float64) < 1 {
			os.Remove(path.Join(torrent.SavePath, file["name"].(string)))
		} else {
			if !slices.Contains(allowedExtensions, path.Ext(file["name"].(string))) {
				continue
			}

			fileName := path.Base(file["name"].(string))
			fileExt := path.Ext(fileName)

			torrentContentFiles = append(torrentContentFiles, file)
			torrentContentNewFileNames = append(
				torrentContentNewFileNames,
				fmt.Sprintf(
					"%s%s",
					utils.BeautifyMovieName(strings.TrimSuffix(fileName, fileExt)),
					fileExt,
				),
			)
		}
	}

	return torrentContentFiles, torrentContentNewFileNames, nil
}

func (instance *Actions) SlackNotify(templateName string, templateVars TorrentsSlackNotificationVars) error {
	templateVars.QBittorrentURL = config.Config.QBittorrentPublicUrl
	templateVars.JellyfinURL = config.Config.JellyfinUrl

	return slack.Client.SendMessage(
		templateName,
		templateVars,
	)
}

func (instance *Actions) JellyfinRename(torrent response.Torrent, torrentContentFiles []map[string]any, torrentContentNewFileNames []string) error {
	var renameFailed bool = false

	for _, file := range torrentContentFiles {
		filePath := path.Dir(file["name"].(string))
		fileName := path.Base(file["name"].(string))
		fileExt := path.Ext(fileName)
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

	var dirNameNew string
	dirPath := filepath.Dir(torrent.FilesPath)
	dirName := filepath.Base(torrent.FilesPath)

	if len(torrentContentNewFileNames) == 1 {
		dirNameNew = strings.TrimSuffix(torrentContentNewFileNames[0], path.Ext(torrentContentNewFileNames[0]))
	} else {
		dirNameNew = utils.BeautifyMovieName(dirName)
	}

	dirPathNew := path.Join(
		dirPath,
		dirNameNew,
	)

	err := os.Rename(torrent.FilesPath, dirPathNew)
	if err != nil {
		renameFailed = true
	}

	jellyfin.Client.RefreshLibrary()
	time.Sleep(2 * time.Second)

	qbittorrent.Client.DeleteTorrentTags(torrent.Hash, []string{"jellyfin:rename=pending"})

	if renameFailed {
		qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:rename=failed"})

		t.Log.Error("Failed to rename torrent files", "action", "jellyfin:rename", "torrent_hash", torrent.Hash)
		return fmt.Errorf("Failed to rename torrent files")
	}

	qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:rename=completed"})
	return nil
}

func (instance *Actions) JellyfinFindSubs(torrent response.Torrent, torrentContentNewFileNames []string) error {
	var subsDownloadedCount int = 0
	var subsAlreadyPresentCount int = 0

	jellyfin.Client.RefreshLibrary()
	time.Sleep(2 * time.Second)

	jellyfinItems, err := jellyfin.Client.GetItems()
	if err != nil {
		t.Log.Error("Failed to get Jellyfin items", "error", err.Error())
		qbittorrent.Client.DeleteTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=pending"})
		qbittorrent.Client.AddTorrentTags(torrent.Hash, []string{"jellyfin:find_subs=failed"})

		return fmt.Errorf("Failed to get Jellyfin items")
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
			err = jellyfin.Client.DownloadSubtitles(item["Id"].(string), config.Config.JellyfinSubtitlesDefaultLanguage)

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

	return nil
}
