package qbittorrent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"common/utils"
	// t "connector-downloader/internal/telemetry"
	"connector-downloader/internal/config"
)

var Client *QBittorrentClient

type QBittorrentClient struct {
	APIBaseURL string
	Client     *http.Client
}

type ClientErrorKind string

type ClientError struct {
	Kind       ClientErrorKind
	Message    string
	StatusCode int
	Body       string
	ParsedJSON any
	Err        error
}

const (
	ClientErrorConnection ClientErrorKind = "connection"
	ClientErrorUpstream   ClientErrorKind = "upstream_api"
)

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

func (instance *QBittorrentClient) Init() {
	instance.APIBaseURL = fmt.Sprintf("%s/api/v2", config.Config.QBittorrentUrl)
	instance.Client = &http.Client{
		Timeout: 10 * time.Second,
	}
}

func (instance *QBittorrentClient) Ping() (string, int, error) {
	req := &utils.Req{Client: instance.Client}

	resp, err := req.GET(
		fmt.Sprintf("%s/app/defaultSavePath", instance.APIBaseURL),
		nil,
		nil,
	)

	if err != nil {
		return "not_ok", 0, err
	}

	return "ok", resp.StatusCode, nil
}

func (instance *QBittorrentClient) ListTorrents() (any, error) {
	req := &utils.Req{Client: instance.Client}

	resp, err := req.GET(
		fmt.Sprintf("%s/torrents/info", instance.APIBaseURL),
		nil,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (instance *QBittorrentClient) GetTorrentContent(torrentHash string) (any, error) {
	req := &utils.Req{Client: instance.Client}

	resp, err := req.GET(
		fmt.Sprintf("%s/torrents/files", instance.APIBaseURL),
		nil,
		map[string]string{
			"hash": torrentHash,
		},
	)

	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (instance *QBittorrentClient) StopTorrent(torrentHash string) error {
	reqParams := url.Values{}
	reqParams.Set("hashes", torrentHash)

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/torrents/stop", instance.APIBaseURL),
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

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
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
		fmt.Sprintf("%s/torrents/add", instance.APIBaseURL),
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

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	if strings.Contains(strings.ToLower(string(body)), "fails") {
		return newUpstreamError("Invalid torrent parameters", 502, body, nil)
	}

	return nil
}

func (instance *QBittorrentClient) RemoveTorrent(torrentHash string, deleteFiles bool) error {
	reqParams := url.Values{}
	reqParams.Set("hashes", torrentHash)
	reqParams.Set("deleteFiles", fmt.Sprintf("%t", deleteFiles))

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/torrents/delete", instance.APIBaseURL),
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

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		body, _ := io.ReadAll(resp.Body)
		return newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	return nil
}

func (instance *QBittorrentClient) AddTorrentTags(torrentHash string, tags []string) error {
	reqParams := url.Values{}
	reqParams.Set("hashes", torrentHash)
	reqParams.Set("tags", strings.Join(tags, ","))

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/torrents/addTags", instance.APIBaseURL),
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

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		body, _ := io.ReadAll(resp.Body)
		return newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	return nil
}

func (instance *QBittorrentClient) DeleteTorrentTags(torrentHash string, tags []string) error {
	reqParams := url.Values{}
	reqParams.Set("hashes", torrentHash)
	reqParams.Set("tags", strings.Join(tags, ","))

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/torrents/removeTags", instance.APIBaseURL),
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

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		body, _ := io.ReadAll(resp.Body)
		return newUpstreamError("qBittorrent returned a non-2xx status", resp.StatusCode, body, nil)
	}

	return nil
}
