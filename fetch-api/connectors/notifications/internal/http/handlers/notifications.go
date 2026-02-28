package handlers

import (
	"errors"

	"connector-notifications/internal/config"
	"connector-notifications/internal/http/dto/request"
	"connector-notifications/internal/http/dto/response"
	"connector-notifications/internal/slack"

	"github.com/gofiber/fiber/v2"
)

func NotifyGrafana(ctx *fiber.Ctx) error {
	var reqPayload request.GrafanaNotificationPayload

	if err := ctx.BodyParser(&reqPayload); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid request payload",
			},
		)
	}

	if reqPayload.Status == "" && len(reqPayload.Alerts) == 0 && reqPayload.Title == "" && reqPayload.Message == "" {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "grafana payload must include status, alerts, or a title/message override",
			},
		)
	}

	result, err := slack.Client.SendTemplateMessage("grafana_alert", reqPayload)
	if err != nil {
		var clientErr *config.ClientError
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
		response.NotificationDeliveryResponse{
			Success:   true,
			Message:   "Grafana notification sent to Slack",
			Channel:   result.Channel,
			Timestamp: result.Timestamp,
		},
	)
}

func NotifyDownloader(ctx *fiber.Ctx) error {
	var reqPayload request.DownloaderNotificationPayload

	if err := ctx.BodyParser(&reqPayload); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid request payload",
			},
		)
	}

	if reqPayload.Event == "" {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "event is required",
			},
		)
	}

	if reqPayload.Title == "" && reqPayload.Message == "" {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "title or message is required",
			},
		)
	}

	result, err := slack.Client.SendTemplateMessage("downloader_event", reqPayload)
	if err != nil {
		var clientErr *config.ClientError
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
		response.NotificationDeliveryResponse{
			Success:   true,
			Message:   "Downloader notification sent to Slack",
			Channel:   result.Channel,
			Timestamp: result.Timestamp,
		},
	)
}
