package qbittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	// t "connector-downloader/src/telemetry"
	"connector-downloader/settings"
	"connector-downloader/src/dto/response"
)

var Client *QBittorrentClient


type QBittorrentClient struct {
	Client *http.Client
}

func (instance *QBittorrentClient) Init() error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	instance.Client = client
	return nil
}

func (instance *QBittorrentClient) Ping() (string, int, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v2/app/defaultSavePath", settings.Config.Url),
		nil,
	)
	if err != nil {
		return "not_ok", 0, err
	}

	resp, err := instance.Client.Do(req)
	if err != nil {
		return "not_ok", 0, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "not_ok", 0, err
	}

	if ! (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return "not_ok", resp.StatusCode, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return "ok", resp.StatusCode, nil
}

func (instance *QBittorrentClient) ListTorrents() ([]response.Torrent, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v2/torrents/info", settings.Config.Url),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := instance.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if ! (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var torrents []response.Torrent
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return torrents, nil
}
