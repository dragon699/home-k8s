package src

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"connector-kube/settings"
	t "connector-kube/src/telemetry"

	"github.com/gofiber/fiber/v2"
)

var app = fiber.New()


func init() {
	app.Use(t.TracingMiddleware())

	LoadKubernetesClient()
	LoadHealthChecker()
	LoadRoutes(app)

	t.Log.Info("Starting Fiber server..",
		"host", settings.Config.ListenHost,
		"port", settings.Config.ListenPort,
		"service", settings.Config.Name,
	)

	go func() {
		if err := app.Listen(
			fmt.Sprintf("%s:%d", settings.Config.ListenHost, settings.Config.ListenPort),
		); err != nil {
			t.Log.Error("Failed to start server", "error", err.Error())
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
