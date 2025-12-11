package src

import (
	"os"
	"time"

	"connector-kube/settings"
	"connector-kube/src/health"
	"connector-kube/src/kubernetes"
	"connector-kube/src/routes"
	t "connector-kube/src/telemetry"

	"github.com/go-co-op/gocron"
	"github.com/gofiber/fiber/v2"
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

	// /list routes
	routes.ListPods(app)
}
