package handlers

import (
	"connector-downloader/settings"
	"connector-downloader/src/dto/response"

	"github.com/gofiber/fiber/v2"
)

func Health(ctx *fiber.Ctx) error {
	return ctx.JSON(
		response.HealthResponse{
			ConnectorName:   settings.Config.Name,
			Healthy:         settings.Config.Healthy,
			HealthLastCheck: *settings.Config.HealthLastCheck,
			HealthNextCheck: *settings.Config.HealthNextCheck,
		},
	)
}

func Ready(ctx *fiber.Ctx) error {
	return ctx.JSON(
		response.ReadyResponse{
			Ready: settings.Config.Healthy,
		},
	)
}
