package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"common/utils"
	"connector-downloader/internal/config"
	"connector-downloader/internal/dto/request"
	"connector-downloader/internal/dto/response"
	"connector-downloader/internal/qbittorrent"

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
		} else {
			torrentMeta.ManagedBy = "qBittorrent"
		}

		statesDownloading := []string{
			"allocating", "downloading", "metaDL", "queuedDL", "stalledDL", "checkingDL", "forcedDL",
		}
		statesPaused := []string{
			"pausedUP", "pausedDL", "stoppedDL",
		}
		statesError := []string{
			"error", "missingFiles",
		}

		if slices.Contains(statesDownloading, torrentData.Status) {
			torrentData.Status = "downloading"
		} else if slices.Contains(statesPaused, torrentData.Status) {
			torrentData.Status = "paused"
		} else if slices.Contains(statesError, torrentData.Status) {
			torrentData.Status = "error"
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

			if !(len(tagParts) > 1) {
				continue
			}

			tagOpParts := strings.Split(tagParts[1], "=")

			if !(len(tagOpParts) > 1) {
				continue
			}

			tagAction := response.TorrentMetaScheduledAction{}

			tagCategory := tagParts[0]
			tagOpName := tagOpParts[0]
			tagOpStatus := tagOpParts[1]

			if tagCategory == "jellyfin" {
				if tagOpName == "rename" {
					switch tagOpStatus {
					case "pending":
						tagAction.Description = "Torrent dir/files will be renamed to match Jellyfin library structure once completed."
					case "completed":
						tagAction.Description = "Torrent dir/files renamed to match Jellyfin library structure."
					case "failed":
						tagAction.Description = "[!] Something went wrong while renaming Torrent's content."
					}
				}

				if tagOpName == "find_subs" {
					switch tagOpStatus {
					case "pending":
						tagAction.Description = "Subtitles will be fetched from OpenSubtitles in Jellyfin for this torrent media."
					case "completed":
						tagAction.Description = "Subtitles in Jellyfin fully fetched."
					case "partially_completed":
						tagAction.Description = "Subtitles fetched in Jellyfin for some of the media."
					case "already_present":
						tagAction.Description = "Subtitles and preferred language already present in Jellyfin for this media."
					case "failed":
						tagAction.Description = "[!] Something went wrong while fetching subtitles from OpenSubtitles in Jellyfin for this torrent media."
					}
				}
			}

			if tagCategory == "slack" {
				if tagOpName == "notify" {
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

func SearchTorrents(ctx *fiber.Ctx) error {
	var searchParams request.SearchTorrentsParams

	if err := ctx.QueryParser(&searchParams); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid query parameters",
			},
		)
	}

	if searchParams.Query == "" {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "query is required",
			},
		)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	reqParams := url.Values{}
	reqParams.Add("q", searchParams.Query)
	reqParams.Add("cat", "200") // Videos only category

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/q.php?%s", config.Config.TPBAPIUrl, reqParams.Encode()),
		nil,
	)
	if err != nil {
		return ctx.Status(500).JSON(
			response.ErrorResponse{
				Error: "Failed to create HTTP request",
			},
		)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ctx.Status(500).JSON(
			response.ErrorResponse{
				Error: "Failed to perform search request to TPB API",
			},
		)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ctx.Status(500).JSON(
			response.ErrorResponse{
				Error: "Failed to read response from TPB API",
			},
		)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return ctx.Status(502).JSON(
			response.ErrorResponse{
				Error:            fmt.Sprintf("Unexpected status code from TPB API: %d", resp.StatusCode),
				UpstreamResponse: map[string]any{"status_code": resp.StatusCode, "body": string(body)},
			},
		)
	}

	var TPBTorrents []response.TPBTorrent
	if err := json.Unmarshal(body, &TPBTorrents); err != nil {
		return ctx.Status(500).JSON(
			response.ErrorResponse{
				Error: "TPB API returned invalid response",
			},
		)
	}

	result := make([]response.Torrent, 0, len(TPBTorrents))

	for torrent := range TPBTorrents {
		id, _ := utils.ToInt(TPBTorrents[torrent].ID)
		magnetURI := fmt.Sprintf(
			"magnet:?xt=urn:btih:%s&dn=%s",
			TPBTorrents[torrent].InfoHash,
			url.QueryEscape(TPBTorrents[torrent].Name),
		)
		leechers, _ := utils.ToInt(TPBTorrents[torrent].Leechers)
		seeders, _ := utils.ToInt(TPBTorrents[torrent].Seeders)
		sizeTotalB, _ := utils.ToInt(TPBTorrents[torrent].Size)
		filesCount, _ := utils.ToInt(TPBTorrents[torrent].NumFiles)
		dateAdded, _ := utils.ToInt(TPBTorrents[torrent].Added)

		torrentData := response.Torrent{
			ID:          id,
			Name:        TPBTorrents[torrent].Name,
			Hash:        TPBTorrents[torrent].InfoHash,
			MagnetURI:   magnetURI,
			Leechers:    leechers,
			Seeders:     seeders,
			SizeTotalMB: utils.BytesToMegabytes(sizeTotalB),
			SizeTotalGB: utils.BytesToGigabytes(sizeTotalB),
			FilesCount:  filesCount,
			DateAdded:   utils.TimeFromUnix(dateAdded),
			IMDB:        TPBTorrents[torrent].IMDB,
		}

		result = append(result, torrentData)
	}

	if len(result) > 0 && result[0].ID == 0 {
		result = []response.Torrent{}
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
	findSubs := false
	notify := false

	if reqPayload.Category == "" {
		reqPayload.Category = "jellyfin"
	}

	if (reqPayload.Category == "jellyfin") && !(slices.Contains(reqPayload.Tags, "jellyfin:rename=pending")) {
		reqPayload.Tags = append(reqPayload.Tags, "jellyfin:rename=pending")
	}

	if len(reqPayload.Tags) == 0 {
		reqPayload.Tags = []string{}
	}

	if reqPayload.SavePath == "" {
		reqPayload.SavePath = config.Config.QBittorrentDefaultSavePath
	}

	if reqPayload.Manage != nil {
		manage = *reqPayload.Manage
	}

	if (manage) && !(slices.Contains(reqPayload.Tags, "fetch-api")) {
		reqPayload.Tags = append(reqPayload.Tags, "fetch-api")
	}

	if reqPayload.FindSubs != nil {
		findSubs = *reqPayload.FindSubs
	}

	if findSubs {
		if reqPayload.Category != "jellyfin" {
			return ctx.Status(400).JSON(
				response.ErrorResponse{
					Error: "find_subs can only be true when `category` is jellyfin",
				},
			)
		}

		reqPayload.Tags = append(reqPayload.Tags, "jellyfin:find_subs=pending")
	}

	if reqPayload.Notify != nil {
		notify = *reqPayload.Notify
	}

	if (notify) && !(slices.Contains(reqPayload.Tags, "slack:notify=pending")) {
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
			Message: "Request sent to qBittorrent!",
		},
	)
}

func AddTorrentTags(ctx *fiber.Ctx) error {
	var reqPayload request.AddTorrentTagsPayload

	if err := ctx.BodyParser(&reqPayload); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid request payload",
			},
		)
	}

	if reqPayload.Hash == "" || len(reqPayload.Tags) == 0 {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "hash and at least one tag in tags are required",
			},
		)
	}

	err := qbittorrent.Client.AddTorrentTags(
		reqPayload.Hash,
		reqPayload.Tags,
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
			Message: "Tag/s added to torrent successfully!",
		},
	)
}

func DeleteTorrentTags(ctx *fiber.Ctx) error {
	var reqPayload request.DeleteTorrentTagsPayload

	if err := ctx.BodyParser(&reqPayload); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid request payload",
			},
		)
	}

	if reqPayload.Hash == "" || len(reqPayload.Tags) == 0 {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "hash and at least one tag in tags are required",
			},
		)
	}

	err := qbittorrent.Client.DeleteTorrentTags(
		reqPayload.Hash,
		reqPayload.Tags,
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
			Message: "Tag/s deleted from torrent successfully!",
		},
	)
}
