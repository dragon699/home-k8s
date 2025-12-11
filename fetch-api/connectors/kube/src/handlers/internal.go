package handlers

import (
	"connector-kube/settings"
	"connector-kube/src/dto/response"

	"github.com/gofiber/fiber/v2"
)


func Health(ctx *fiber.Ctx) error {
	return ctx.JSON(
		response.Health{
			ConnectorName:  settings.Config.Name,
			Healthy:        settings.Config.Healthy,
		},
	)
}


func Ready(ctx *fiber.Ctx) error {
	return ctx.JSON(
		response.Ready{
			Ready: settings.Config.Healthy,
		},
	)
}
