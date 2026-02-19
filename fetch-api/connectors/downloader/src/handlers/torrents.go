package handlers

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"common/utils"
	"connector-downloader/settings"
	"connector-downloader/src/dto/request"
	"connector-downloader/src/dto/response"
	"connector-downloader/src/qbittorrent"

	"github.com/gofiber/fiber/v2"
)

func ListTorrents(ctx *fiber.Ctx) error {
	torrents, err := qbittorrent.Client.ListTorrents()

	if err != nil {
		var clientErr *qbittorrent.ClientError
		if errors.As(err, &clientErr) {
			return ctx.Status(502).JSON(
				response.ErrorResponse{
					Error:            err.Error(),
					UpstreamResponse: clientErr.UpstreamResponse(),
				},
			)
		}

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

			if (torrentData.Category == "jellyfin") && ! (utils.HasItemWithPrefix(torrentData.Tags, "jellyfin:")) {
				qbittorrent.Client.AddTorrentTags(
					torrentData.Hash,
					[]string{"jellyfin:rename=pending"},
				)
				torrentData.Tags = append(torrentData.Tags, "jellyfin:rename=pending")
			}
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
			tagOpName := tagOpParts[0]
			tagOpStatus := tagOpParts[1]

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

			if (tagCategory == "slack") && (tagOpName == "notify") {
				switch tagOpStatus {
				case "pending":
					tagAction.Description = "Slack notification still not sent."
				case "initial":
					tagAction.Description = "Initial Slack notification already sent, awaiting for torrent completion."
				case "completed":
					tagAction.Description = "Slack notifications sent."
				case "failed":
					tagAction.Description = "[!] Something went wrong while sending notifications to Slack."
				}
			}

			tagAction.Name = tagOpName
			tagAction.Status = tagOpStatus
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

func AddTorrent(ctx *fiber.Ctx) error {
	var reqPayload request.AddTorrentPayload

	if err := ctx.BodyParser(&reqPayload); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid request payload",
			},
		)
	}

	if reqPayload.URL == "" {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "url is required",
			},
		)
	}

	manage := true
	notify := false

	if reqPayload.Category == "" {
		reqPayload.Category = "jellyfin"
	}

	if len(reqPayload.Tags) == 0 {
		reqPayload.Tags = []string{}
	}

	if reqPayload.SavePath == "" {
		reqPayload.SavePath = settings.Config.QBittorrentDefaultSavePath
	}

	if reqPayload.Manage != nil {
		manage = *reqPayload.Manage
	}

	if (manage) && ! (slices.Contains(reqPayload.Tags, "fetch-api")) {
		reqPayload.Tags = append(reqPayload.Tags, "fetch-api")
	}

	if reqPayload.Notify != nil {
		notify = *reqPayload.Notify
	}

	if (notify) && ! (slices.Contains(reqPayload.Tags, "slack:notify=pending")) {
		reqPayload.Tags = append(reqPayload.Tags, "slack:notify=pending")
	}

	err := qbittorrent.Client.AddTorrent(
		reqPayload.URL,
		reqPayload.Category,
		reqPayload.Tags,
		reqPayload.SavePath,
	)

	if err != nil {
		var clientErr *qbittorrent.ClientError
		if errors.As(err, &clientErr) {
			return ctx.Status(502).JSON(
				response.ErrorResponse{
					Error:            err.Error(),
					UpstreamResponse: clientErr.UpstreamResponse(),
				},
			)
		}

		return ctx.Status(500).JSON(
			response.ErrorResponse{
				Error: err.Error(),
			},
		)
	}

	return ctx.JSON(
		response.SuccessResponse{
			Success: true,
			Message: fmt.Sprintf("Torrent added successfully and will begin downloading shortly -> %s", settings.Config.QBittorrentPublicUrl),
		},
	)
}
