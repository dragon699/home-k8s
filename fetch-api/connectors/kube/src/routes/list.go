package routes

import (
	"connector-kube/src/handlers"

	"github.com/gofiber/fiber/v2"
)

const listRouterName = "/list"


func ListPods(router fiber.Router) {
	api := router.Group(listRouterName)
	api.Get("/pods", handlers.ListPods)
}
