package handlers

import (
	"slices"
	"strings"

	"common/utils"
	"connector-downloader/src/dto/response"
	"connector-downloader/src/qbittorrent"

	"github.com/gofiber/fiber/v2"
)


func ListTorrents(ctx *fiber.Ctx) error {
	torrents, err := qbittorrent.Client.ListTorrents()

	if err != nil {
		return ctx.Status(500).JSON(
			response.ErrorResponse{
				Error: err.Error(),
			},
		)
	}

	result := make([]response.Torrent, 0, len(torrents))

	for _, torrent := range torrents {
		torrentSrc, ok := torrent.(map[string]any)
		if !ok {
			continue
		}

		torrentData := response.Torrent{
			Name:               torrentSrc["name"].(string),
			Hash:               torrentSrc["hash"].(string),
			Category:           torrentSrc["category"].(string),
			Tags:               strings.Split(torrentSrc["tags"].(string), ", "),
			Status:             torrentSrc["state"].(string),
			ProgressPercentage: utils.ProgressToPercentage(torrentSrc["progress"].(float64)),
			EtaMinutes:         utils.SecondsToMinutes(int64(torrentSrc["eta"].(float64))),
			MagnetURI:          torrentSrc["magnet_uri"].(string),
			Leechers:           int64(torrentSrc["num_leechs"].(float64)),
			Seeders:            int64(torrentSrc["num_seeds"].(float64)),

			DateAdded:        utils.TimeFromUnix(int64(torrentSrc["added_on"].(float64))),
			DateLastActivity: utils.TimeFromUnix(int64(torrentSrc["last_activity"].(float64))),
			DateCompleted:    utils.TimeFromUnix(int64(torrentSrc["completion_on"].(float64))),

			SizeTotalMB:      utils.BytesToMegabytes(int64(torrentSrc["total_size"].(float64))),
			SizeDownloadedMB: utils.BytesToMegabytes(int64(torrentSrc["downloaded"].(float64))),
			SizeUploadedMB:   utils.BytesToMegabytes(int64(torrentSrc["uploaded"].(float64))),
			SizeLeftMB:       utils.BytesToMegabytes(int64(torrentSrc["amount_left"].(float64))),
			SizeMB:           utils.BytesToMegabytes(int64(torrentSrc["size"].(float64))),

			FilesAvailabilityPercentage: utils.RoundToTwoDecimals(torrentSrc["availability"].(float64) * 100),
			FilesPath:                   torrentSrc["content_path"].(string),
			SavePath:                    torrentSrc["save_path"].(string),

			SpeedDownloadMBps: utils.BytesPerSecondToMBps(int64(torrentSrc["dlspeed"].(float64))),
			SpeedUploadMBps:   utils.BytesPerSecondToMBps(int64(torrentSrc["upspeed"].(float64))),
		}

		torrentMeta := response.TorrentMeta{}

		if slices.Contains(torrentData.Tags, "fetch-api") {
			torrentMeta.ManagedBy = "connector-downloader"
		} else {
			torrentMeta.ManagedBy = "qBittorrent"
		}

		statesDownloading := []string{"allocating", "downloading", "metaDL", "queuedDL", "stalledDL", "checkingDL", "forcedDL"}
		statesPaused := []string{"pausedUP", "pausedDL"}

		if slices.Contains(statesDownloading, torrentData.Status) {
			torrentData.Status = "downloading"
		} else if slices.Contains(statesPaused, torrentData.Status) {
			torrentData.Status = "paused"
		} else {
			torrentData.Status = "unknown"
		}

		if torrentData.ProgressPercentage == 100 {
			torrentData.Status = "completed"
			torrentData.EtaMinutes = 0
		} else {
			torrentData.DateCompleted = ""
		}

		if (torrentData.Category == "jellyfin") && ! (utils.HasItemWithPrefix(torrentData.Tags, "jellyfin:")) {
			qbittorrent.Client.AddTorrentTags(
				torrentData.Hash,
				[]string{
					"jellyfin:pending=rename",
				},
			)
			torrentData.Tags = append(torrentData.Tags, "jellyfin:pending=rename")
		}

		for _, tag := range torrentData.Tags {
			tagParts := strings.Split(tag, ":")

			if ! (len(tagParts) > 1) {
				continue
			}

			tagOpParts := strings.Split(tagParts[1], "=")

			if ! (len(tagOpParts) > 1) {
				continue
			}

			tagAction := response.TorrentMetaScheduledAction{}

			tagCategory := tagParts[0]
			tagOpStatus := tagOpParts[0]
			tagOpName   := tagOpParts[1]

			if (tagCategory == "jellyfin") && (tagOpName == "rename") {
				switch tagOpStatus {
				case "pending":
					tagAction.Description = "Torrent dir/files will be renamed to match Jellyfin library structure once completed."
				case "completed":
					tagAction.Description = "Torrent dir/files renamed to match Jellyfin library structure."
				case "failed":
					tagAction.Description = "[!] Something went wrong while renaming Torrent's content."
				}
			}

			tagAction.Name =     tagOpName
			tagAction.Status =   tagOpStatus
			tagAction.Category = tagCategory

			if (tagAction.Name != "") && (tagAction.Status != "") {
				torrentMeta.ScheduledActions = append(torrentMeta.ScheduledActions, tagAction)
			}
		}

		torrentData.Meta = torrentMeta
		result = append(result, torrentData)
	}

	return ctx.JSON(response.BaseResponse[response.Torrent]{
		TotalItems: len(result),
		Items:      result,
	})
}
