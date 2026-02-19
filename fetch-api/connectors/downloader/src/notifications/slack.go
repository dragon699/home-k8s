package notifications

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)


type NotificationTorrentsVars struct {
	TorrentName  	string  `json:"torrent_name"`
	QBittorrentURL  string  `json:"qbittorrent_url"`
	JellyfinURL     string  `json:"jellyfin_url,omitempty"`
}


func SendSlackNotification(url string, payload string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		strings.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("Failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to send HTTP request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read HTTP response: %w", err)
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return fmt.Errorf("Slack returned a non-2xx status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
