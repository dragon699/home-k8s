package handlers

import (
	"errors"
	"fmt"

	"connector-slack/internal/config"
	"connector-slack/internal/http/dto/request"
	"connector-slack/internal/http/dto/response"
	"connector-slack/internal/slack"

	"github.com/gofiber/fiber/v2"
	slackapi "github.com/slack-go/slack"
)

func SendNotification(ctx *fiber.Ctx) error {
	var reqPayload request.NotificationPayload

	if err := ctx.BodyParser(&reqPayload); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid request payload",
			},
		)
	}

	if len(reqPayload.Blocks) == 0 {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Got 0 blocks in payload, at least 1 is required",
			},
		)
	}

	options := []slackapi.MsgOption{
		slackapi.MsgOptionUsername(reqPayload.Options.User),
		slackapi.MsgOptionIconURL(reqPayload.Options.UserIcon),
		slackapi.MsgOptionText(reqPayload.Options.ExtraText, false),
	}

	result, err := slack.Client.SendMsg(reqPayload.ChannelID, reqPayload.Blocks, options...)

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
		response.NotificationStatus{
			Success:   true,
			Channel:   result.Channel,
			Timestamp: result.Timestamp,
		},
	)
}

func SendGrafanaAlertNotification(ctx *fiber.Ctx) error {
	var reqPayload request.GrafanaAlertNotificationPayload

	if err := ctx.BodyParser(&reqPayload); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid request payload",
			},
		)
	}

	if len(reqPayload.Alerts) == 0 {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Got 0 alerts in payload, at least 1 is required",
			},
		)
	}

	result, err := slack.Client.SendMsgFromTemplate("grafana/alert", reqPayload)

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
		response.NotificationStatus{
			Success:   true,
			Channel:   result.Channel,
			Timestamp: result.Timestamp,
		},
	)
}

func SendTorrentNotification(ctx *fiber.Ctx) error {
	var reqPayload request.TorrentNotificationPayload

	if err := ctx.BodyParser(&reqPayload); err != nil {
		return ctx.Status(400).JSON(
			response.ErrorResponse{
				Error: "Invalid request payload",
			},
		)
	}

	fmt.Println(reqPayload)

	result, err := slack.Client.SendMsgFromTemplate("connector-downloader/torrent", reqPayload)

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
		response.NotificationStatus{
			Success:   true,
			Channel:   result.Channel,
			Timestamp: result.Timestamp,
		},
	)
}
