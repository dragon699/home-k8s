package handlers

import (
	"connector-kube/settings"
	"connector-kube/src/dto/response"

	"github.com/gofiber/fiber/v2"
)


// Health godoc
// @Summary      Health check
// @Tags         health
// @Produce      json
// @Success      200  {object}  response.HealthResponse
// @Router       /api/health [get]
func Health(ctx *fiber.Ctx) error {
	return ctx.JSON(
		response.HealthResponse{
			ConnectorName:  settings.Config.Name,
			Healthy:        settings.Config.Healthy,
			HealthLastCheck: *settings.Config.HealthLastCheck,
			HealthNextCheck: *settings.Config.HealthNextCheck,
		},
	)
}


// Ready godoc
// @Summary      Readiness check
// @Tags         health
// @Produce      json
// @Success      200  {object}  response.ReadyResponse
// @Router       /api/ready [get]
func Ready(ctx *fiber.Ctx) error {
	return ctx.JSON(
		response.ReadyResponse{
			Ready: settings.Config.Healthy,
		},
	)
}
