package handlers

import (
	"errors"

	"connector-slack/internal/config"
	"connector-slack/internal/http/dto/request"
	"connector-slack/internal/http/dto/response"
	"connector-slack/internal/slack"

	"github.com/gofiber/fiber/v2"
)

func SlackGrafana(ctx *fiber.Ctx) error {
	var reqPayload request.GrafanaSlackPayload

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
		response.SlackDeliveryResponse{
			Success:   true,
			Message:   "Grafana notification sent to Slack",
			Channel:   result.Channel,
			Timestamp: result.Timestamp,
		},
	)
}

func SlackDownloader(ctx *fiber.Ctx) error {
	var reqPayload request.DownloaderSlackPayload

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
		response.SlackDeliveryResponse{
			Success:   true,
			Message:   "Downloader notification sent to Slack",
			Channel:   result.Channel,
			Timestamp: result.Timestamp,
		},
	)
}
