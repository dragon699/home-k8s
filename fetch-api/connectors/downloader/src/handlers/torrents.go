package handlers

import (
	"connector-downloader/src/dto/response"
	"connector-downloader/src/qbittorrent"

	"github.com/gofiber/fiber/v2"
)


func ListTorrents(ctx *fiber.Ctx) error {
	result, err := qbittorrent.Client.ListTorrents()

	if err != nil {
		return ctx.Status(500).JSON(
			response.ErrorResponse{
				Error: err.Error(),
			},
		)
	}

	return ctx.JSON(response.BaseResponse[response.Torrent]{
		TotalItems: len(result),
		Items:      result,
	})
}
