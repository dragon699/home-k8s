package src

import (
	"fmt"
	"os"
	"time"

	docs "connector-downloader/docs"
	"connector-downloader/settings"
	"connector-downloader/src/health"
	"connector-downloader/src/qbittorrent"
	"connector-downloader/src/routes"
	"connector-downloader/src/swagger"
	t "connector-downloader/src/telemetry"
	action_scheduler "connector-downloader/src/torrent_actions"

	"github.com/go-co-op/gocron"
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

var scheduler = gocron.NewScheduler(time.UTC)
var healthChecker = health.HealthChecker{
	Scheduler: scheduler,
}
var actionChecker = action_scheduler.ActionChecker{
	Scheduler: scheduler,
}

func LoadQBittorrentClient() {
	qbittorrent.Client = &qbittorrent.QBittorrentClient{}
	err := qbittorrent.Client.Init()

	if err != nil {
		t.Log.Error("Failed to create QBittorrent client", "error", err.Error())
		os.Exit(1)
	}
}

func LoadHealthChecker() {
	healthChecker.CreateSchedule()
	scheduler.StartAsync()
}

func LoadActionChecker() {
	actionChecker.CreateSchedule()
}

func LoadRoutes(app fiber.Router) {
	// /api routes
	routes.Health(app)
	routes.Ready(app)

	// /torrents routes
	routes.ListTorrents(app)
	routes.AddTorrent(app)

	// Swagger routes
	docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", settings.Config.ListenHost, settings.Config.ListenPort)

	if settings.Config.OtelServiceVersion != "" {
		docs.SwaggerInfo.Version = settings.Config.OtelServiceVersion
	}

	swaggerHandler := fiberSwagger.FiberWrapHandler(
		fiberSwagger.URL("/swagger/doc.json"),
		fiberSwagger.DocExpansion("list"),
		fiberSwagger.DeepLinking(true),
		fiberSwagger.PersistAuthorization(true),
	)
	app.Get("/swagger/*", swaggerHandler)
	app.Get("/docs", swagger.Handler("/swagger/doc.json"))
}
