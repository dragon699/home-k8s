package qbittorrent

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func (instance *QBittorrentClient) ListTorrents() ([]map[string]any, error) {
	req := &utils.Req{Client: instance.Client}

	resp, err := req.GET(
		fmt.Sprintf("%s/torrents/info", instance.APIBaseURL),
		nil,
		nil,
	)

	if err != nil {
		return nil, err
	}

	return resp.Body.([]map[string]any), nil
}

func (instance *QBittorrentClient) GetTorrentContent(torrentHash string) ([]map[string]any, error) {
	req := &utils.Req{Client: instance.Client}

	resp, err := req.GET(
		fmt.Sprintf("%s/torrents/files", instance.APIBaseURL),
		nil,
		map[string]any{
			"hash": torrentHash,
		},
	)

	if err != nil {
		return nil, err
	}

	return resp.Body.([]map[string]any), nil
}

func (instance *QBittorrentClient) StopTorrent(torrentHash string) error {
	req := &utils.Req{Client: instance.Client}

	_, err := req.POST(
		fmt.Sprintf("%s/torrents/stop", instance.APIBaseURL),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		map[string]any{
			"hashes": torrentHash,
		},
		nil,
	)

	if err != nil {
		return err
	}

	return nil
}

func (instance *QBittorrentClient) AddTorrent(torrentURL string, category string, tags []string, savePath string) error {
	req := &utils.Req{Client: instance.Client}

	resp, err := req.POST(
		fmt.Sprintf("%s/torrents/add", instance.APIBaseURL),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		map[string]any{
			"urls":     torrentURL,
			"savepath": savePath,
			"category": category,
			"tags":     strings.Join(tags, ","),
		},
		nil,
	)

	if err != nil {
		return err
	}

	if strings.Contains(strings.ToLower(fmt.Sprintf("%v", resp.Body)), "fails") {
		return newUpstreamError("Invalid torrent parameters", 502, []byte(fmt.Sprintf("%v", resp.Body)), nil)
	}

	return nil
}

func (instance *QBittorrentClient) RemoveTorrent(torrentHash string, deleteFiles bool) error {
	req := &utils.Req{Client: instance.Client}

	_, err := req.POST(
		fmt.Sprintf("%s/torrents/delete", instance.APIBaseURL),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		map[string]any{
			"hashes":      torrentHash,
			"deleteFiles": deleteFiles,
		},
		nil,
	)

	if err != nil {
		return err
	}

	return nil
}

func (instance *QBittorrentClient) AddTorrentTags(torrentHash string, tags []string) error {
	req := &utils.Req{Client: instance.Client}

	_, err := req.POST(
		fmt.Sprintf("%s/torrents/addTags", instance.APIBaseURL),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		map[string]any{
			"hashes": torrentHash,
			"tags":   strings.Join(tags, ","),
		},
		nil,
	)

	if err != nil {
		return err
	}

	return nil
}

func (instance *QBittorrentClient) DeleteTorrentTags(torrentHash string, tags []string) error {
	req := &utils.Req{Client: instance.Client}

	_, err := req.POST(
		fmt.Sprintf("%s/torrents/removeTags", instance.APIBaseURL),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		map[string]any{
			"hashes": torrentHash,
			"tags":   strings.Join(tags, ","),
		},
		nil,
	)

	if err != nil {
		return err
	}

	return nil
}
