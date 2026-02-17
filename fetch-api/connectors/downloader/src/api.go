// connector-downloader bootstrap
//
// @title           Downloader Connector
// @description     A connector that downloads data from various sources to the host.
// @BasePath        /
// @produce         json
package src

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"connector-downloader/settings"
	t "connector-downloader/src/telemetry"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

var app = fiber.New(fiber.Config{
	AppName: "connector-downloader",
})

func init() {
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))

	app.Use(t.TracingMiddleware())

	LoadQBittorrentClient()
	LoadHealthChecker()
	LoadActionChecker()
	LoadRoutes(app)

	t.Log.Info("Starting Fiber..",
		"host", settings.Config.ListenHost,
		"port", settings.Config.ListenPort,
		"service", settings.Config.Name,
	)

	go func() {
		if err := app.Listen(
			fmt.Sprintf("%s:%d", settings.Config.ListenHost, settings.Config.ListenPort),
		); err != nil {
			t.Log.Error("Failed to start Fiber", "error", err.Error())
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if err := app.Shutdown(); err != nil {
		t.Log.Error("Failed to shutdown server", "error", err.Error())
	}
}
