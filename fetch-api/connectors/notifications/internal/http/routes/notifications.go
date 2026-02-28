package routes

import (
	_ "notifications-controller/internal/http/dto/request"
	_ "notifications-controller/internal/http/dto/response"
	"notifications-controller/internal/http/handlers"

	"github.com/gofiber/fiber/v2"
)

const notificationsRouterName = "/notifications"

// NotifyGrafana godoc
// @Summary      Forward a Grafana alert payload to Slack
// @Description  Accepts a Grafana Alerting webhook payload and posts it to Slack.
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request  body      request.GrafanaNotificationPayload  true  "Grafana notification payload"
// @Success      200      {object}  response.NotificationDeliveryResponse
// @Failure      400      {object}  response.ErrorResponse
// @Failure      500      {object}  response.ErrorResponse
// @Router       /notifications/grafana [post]
func NotifyGrafana(router fiber.Router) {
	api := router.Group(notificationsRouterName)
	api.Post("/grafana", handlers.NotifyGrafana)
}

// NotifyDownloader godoc
// @Summary      Forward a connector-downloader event payload to Slack
// @Description  Accepts a downloader-originated notification payload and posts it to Slack.
// @Tags         notifications
// @Accept       json
// @Produce      json
// @Param        request  body      request.DownloaderNotificationPayload  true  "Downloader notification payload"
// @Success      200      {object}  response.NotificationDeliveryResponse
// @Failure      400      {object}  response.ErrorResponse
// @Failure      500      {object}  response.ErrorResponse
// @Router       /notifications/downloader [post]
func NotifyDownloader(router fiber.Router) {
	api := router.Group(notificationsRouterName)
	api.Post("/downloader", handlers.NotifyDownloader)
}
