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

type ClientErrorKind string

const (
	ClientErrorConnection ClientErrorKind = "connection"
	ClientErrorUpstream   ClientErrorKind = "upstream_api"
)

type ClientError struct {
	Kind       ClientErrorKind
	Message    string
	StatusCode int
	Body       string
	ParsedJSON any
	Err        error
}

func (e *ClientError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ClientError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *ClientError) UpstreamResponse() map[string]any {
	if e == nil {
		return nil
	}

	resp := map[string]any{
		"error_type": string(e.Kind),
	}

	if e.StatusCode > 0 {
		resp["status_code"] = e.StatusCode
	}

	if e.Body != "" {
		resp["body"] = e.Body
	}

	if e.ParsedJSON != nil {
		resp["json"] = e.ParsedJSON
	}

	return resp
}

func parseJSONBody(body []byte) any {
	var parsed any
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}
	return parsed
}

func newConnectionError(message string, err error) error {
	return &ClientError{
		Kind:    ClientErrorConnection,
		Message: message,
		Err:     err,
	}
}

func newUpstreamError(message string, statusCode int, body []byte, err error) error {
	return &ClientError{
		Kind:       ClientErrorUpstream,
		Message:    message,
		StatusCode: statusCode,
		Body:       string(body),
		ParsedJSON: parseJSONBody(body),
		Err:        err,
	}
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

	if ! (resp.StatusCode >= 200) && (resp.StatusCode < 300) {
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
		return nil, newConnectionError("Failed to create HTTP request", err)
	}

	resp, err := instance.Client.Do(req)
	if err != nil {
		return nil, newConnectionError("Failed to create HTTP request", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newConnectionError("Failed to read HTTP response", err)
	}

	if ! (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	var torrents []any
	if err := json.Unmarshal(body, &torrents); err != nil {
		return nil, newUpstreamError("qBittorrent returned an invalid response", resp.StatusCode, body, err)
	}

	return torrents, nil
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
		return nil, newConnectionError("Failed to create HTTP request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return nil, newConnectionError("Failed to create HTTP request", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, newConnectionError("Failed to read HTTP response", err)
	}

	if ! (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	var files []map[string]any
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return files, nil
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
		return newConnectionError("Failed to create HTTP request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return newConnectionError("Failed to create HTTP request", err)
	}

	defer resp.Body.Close()

	if ! (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		body, _ := io.ReadAll(resp.Body)
		return newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	return nil
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
		return newConnectionError("Failed to create HTTP request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return newConnectionError("Failed to create HTTP request", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return newConnectionError("Failed to read HTTP response", err)
	}

	if ! (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	if strings.Contains(strings.ToLower(string(body)), "fails") {
		return newUpstreamError("Invalid torrent parameters", 502, body, nil)
	}

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
		return newConnectionError("Failed to create HTTP request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return newConnectionError("Failed to create HTTP request", err)
	}

	defer resp.Body.Close()

	if ! (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		body, _ := io.ReadAll(resp.Body)
		return newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

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
		return newConnectionError("Failed to create HTTP request", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := instance.Client.Do(req)
	if err != nil {
		return newConnectionError("Failed to create HTTP request", err)
	}

	defer resp.Body.Close()

	if ! (resp.StatusCode >= 200 && resp.StatusCode < 300) {
		body, _ := io.ReadAll(resp.Body)
		return newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	return nil
}
