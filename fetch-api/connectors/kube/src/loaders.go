package src

import (
	"fmt"
	"os"
	"time"

	docs "connector-kube/docs"
	"connector-kube/settings"
	"connector-kube/src/health"
	"connector-kube/src/kubernetes"
	"connector-kube/src/routes"
	"connector-kube/src/swagger"
	t "connector-kube/src/telemetry"

	"github.com/go-co-op/gocron"
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

var scheduler = gocron.NewScheduler(time.UTC)
var healthChecker = health.HealthChecker{
	Scheduler: scheduler,
}


func LoadKubernetesClient() {
	kubernetes.Client = &kubernetes.KubernetesClient{
		InCluster:      settings.Config.InCluster,
		KubeConfigPath: settings.Config.KubeConfigPath,
	}
	err := kubernetes.Client.Init()

	if err != nil {
		t.Log.Error("Failed to create Kubernetes client", "error", err.Error())
		os.Exit(1)
	}
}


func LoadHealthChecker() {
	healthChecker.CreateSchedule()
	scheduler.StartAsync()
}


func LoadRoutes(app fiber.Router) {
	// /api routes
	routes.Health(app)
	routes.Ready(app)

	// /pods routes
	routes.ListPods(app)

	// /workloads/* routes
	routes.ListDeployments(app)

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
