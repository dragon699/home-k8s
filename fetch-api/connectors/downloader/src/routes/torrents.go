package routes

import (
	"connector-downloader/src/handlers"

	"github.com/gofiber/fiber/v2"
)

const torrentsRouterName = "/torrents"

// ListTorrents godoc
// @Summary      List torrents in qBittorrent
// @Description  Returns list of torrents in qBittorrent.
// @Tags         torrents
// @Produce      json
// @Success      200  {object}  response.TorrentListResponse
// @Failure      400  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /torrents [get]
func ListTorrents(router fiber.Router) {
	api := router.Group(torrentsRouterName)
	api.Get("/", handlers.ListTorrents)
}

// Search godoc
// @Summary      Search for torrents in torrent providers
// @Description  Searches for torrents in torrent providers.
// @Tags         torrents
// @Accept       json
// @Produce      json
// @Param        request  query     request.SearchTorrentsParams  true  "Search torrents params"
// @Success      200      {object}  response.TorrentListResponse
// @Failure      400      {object}  response.ErrorResponse
// @Failure      500      {object}  response.ErrorResponse
// @Router       /torrents/search [get]
func SearchTorrents(router fiber.Router) {
	api := router.Group(torrentsRouterName)
	api.Get("/search", handlers.SearchTorrents)
}

// AddTorrent godoc
// @Summary      Add a torrent to qBittorrent
// @Description  Adds a torrent to qBittorrent using a torrent or magnet url.
// @Tags         torrents
// @Accept       json
// @Produce      json
// @Param        request  body      request.AddTorrentPayload  true  "Torrent payload"
// @Success      200      {object}  response.SuccessResponse
// @Failure      400      {object}  response.ErrorResponse
// @Failure      500      {object}  response.ErrorResponse
// @Router       /torrents [post]
func AddTorrent(router fiber.Router) {
	api := router.Group(torrentsRouterName)
	api.Post("/", handlers.AddTorrent)
}
