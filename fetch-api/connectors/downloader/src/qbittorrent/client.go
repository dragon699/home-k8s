package qbittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	// t "connector-downloader/src/telemetry"
	"connector-downloader/settings"
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
		fmt.Sprintf("%s/api/v2/app/defaultSavePath", settings.Config.QBittorrentUrl),
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

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return "not_ok", resp.StatusCode, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return "ok", resp.StatusCode, nil
}

func (instance *QBittorrentClient) ListTorrents() ([]any, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v2/torrents/info", settings.Config.QBittorrentUrl),
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

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var torrents []any
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return torrents, nil
}

func (instance *QBittorrentClient) AddTorrent(torrentURL string, category string, tags []string, savePath string) error {
	reqParams := url.Values{}
	reqParams.Set("urls", torrentURL)
	reqParams.Set("savepath", savePath)
	reqParams.Set("category", category)
	reqParams.Set("tags", strings.Join(tags, ","))

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v2/torrents/add", settings.Config.QBittorrentUrl),
		strings.NewReader(reqParams.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func (instance *QBittorrentClient) AddTorrentTags(torrent_hash string, tags []string) error {
	reqParams := url.Values{}
	reqParams.Set("hashes", torrent_hash)
	reqParams.Set("tags", strings.Join(tags, ","))

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v2/torrents/addTags", settings.Config.QBittorrentUrl),
		strings.NewReader(reqParams.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func (instance *QBittorrentClient) RemoveTorrentTags(torrent_hash string, tags []string) error {
	reqParams := url.Values{}
	reqParams.Set("hashes", torrent_hash)
	reqParams.Set("tags", strings.Join(tags, ","))

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v2/torrents/removeTags", settings.Config.QBittorrentUrl),
		strings.NewReader(reqParams.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func (instance *QBittorrentClient) StopTorrent(torrent_hash string) error {
	reqParams := url.Values{}
	reqParams.Set("hashes", torrent_hash)

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v2/torrents/stop", settings.Config.QBittorrentUrl),
		strings.NewReader(reqParams.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func (instance *QBittorrentClient) GetTorrentContent(torrent_hash string) ([]map[string]any, error) {
	reqParams := url.Values{}
	reqParams.Set("hash", torrent_hash)

	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v2/torrents/files?%s", settings.Config.QBittorrentUrl, reqParams.Encode()),
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var files []map[string]any
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return files, nil
}
