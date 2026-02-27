package jellyfin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"common/utils"
	"connector-downloader/internal/config"
)

var Client *JellyfinClient

type JellyfinClient struct {
	APIBaseURL        string
	APIDefaultHeaders map[string]string
	Client            *http.Client
}

func (instance *JellyfinClient) Init() {
	instance.APIBaseURL = config.Config.JellyfinUrl
	instance.APIDefaultHeaders = map[string]string{
		"Content-Type": "application/json",
		"X-Emby-Token": config.Config.JellyfinAPIKey,
	}
	instance.Client = &http.Client{
		Timeout: 10 * time.Second,
	}
}

func (instance *JellyfinClient) RefreshLibrary() error {
	req := &utils.Req{Client: instance.Client}

	_, err := req.POST(
		fmt.Sprintf("%s/Library/Refresh", instance.APIBaseURL),
		instance.APIDefaultHeaders,
		nil,
		nil,
	)

	return err
}

func (instance *JellyfinClient) GetItems() ([]map[string]any, error) {
	req := &utils.Req{Client: instance.Client}

	resp, err := req.GET(
		fmt.Sprintf("%s/Items", instance.APIBaseURL),
		instance.APIDefaultHeaders,
		map[string]any{
			"Recursive":        true,
			"Fields":           "Path,DateCreated,MediaStreams",
			"IncludeItemTypes": "Episode,Movie,Series,Trailer,Video",
			"sortBy":           "DateCreated",
			"sortOrder":        "Descending",
		},
	)

	if err != nil {
		return nil, err
	}

	result := struct {
		Items []map[string]any `json:"Items"`
	}{}

	rawBody, err := json.Marshal(resp.Body)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, err
	}

	return result.Items, nil
}
