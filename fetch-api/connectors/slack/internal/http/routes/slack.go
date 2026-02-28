package routes

import (
	_ "connector-slack/internal/http/dto/request"
	_ "connector-slack/internal/http/dto/response"
	"connector-slack/internal/http/handlers"

	"github.com/gofiber/fiber/v2"
)

const slackRouterName = "/slack"

// SlackGrafana godoc
// @Summary      Forward a Grafana alert payload to Slack
// @Description  Accepts a Grafana Alerting webhook payload and posts it to Slack.
// @Tags         slack
// @Accept       json
// @Produce      json
// @Param        request  body      request.GrafanaSlackPayload  true  "Grafana Slack payload"
// @Success      200      {object}  response.SlackDeliveryResponse
// @Failure      400      {object}  response.ErrorResponse
// @Failure      500      {object}  response.ErrorResponse
// @Router       /slack/grafana [post]
func SlackGrafana(router fiber.Router) {
	api := router.Group(slackRouterName)
	api.Post("/grafana", handlers.SlackGrafana)
}

// SlackDownloader godoc
// @Summary      Forward a connector-downloader event payload to Slack
// @Description  Accepts a downloader-originated Slack payload and posts it to Slack.
// @Tags         slack
// @Accept       json
// @Produce      json
// @Param        request  body      request.DownloaderSlackPayload  true  "Downloader Slack payload"
// @Success      200      {object}  response.SlackDeliveryResponse
// @Failure      400      {object}  response.ErrorResponse
// @Failure      500      {object}  response.ErrorResponse
// @Router       /slack/downloader [post]
func SlackDownloader(router fiber.Router) {
	api := router.Group(slackRouterName)
	api.Post("/downloader", handlers.SlackDownloader)
}
