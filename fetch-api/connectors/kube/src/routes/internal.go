package routes

import (
	"connector-kube/src/handlers"

	"github.com/gofiber/fiber/v2"
)

const apiRouterName = "/api"


func Health(router fiber.Router) {
	api := router.Group(apiRouterName)
	api.Get("/health", handlers.Health)
}

func Ready(router fiber.Router) {
	api := router.Group(apiRouterName)
	api.Get("/ready", handlers.Ready)
}
