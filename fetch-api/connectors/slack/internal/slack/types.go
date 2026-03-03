package slack

import (
	"encoding/json"

	slackapi "github.com/slack-go/slack"
)

type Message struct {
	Channel  string           `json:"channel"`
	Username string           `json:"username,omitempty"`
	IconURL  string           `json:"icon_url,omitempty"`
	Text     string           `json:"text,omitempty"`
	Blocks   []map[string]any `json:"blocks,omitempty"`
}

type Response struct {
	OK        bool
	Channel   string
	Timestamp string
}

type rawBlock struct {
	data map[string]any
}

func (b rawBlock) BlockType() slackapi.MessageBlockType {
	if t, ok := b.data["type"].(string); ok {
		return slackapi.MessageBlockType(t)
	}
	return ""
}

func (b rawBlock) ID() string {
	if id, ok := b.data["block_id"].(string); ok {
		return id
	}
	return ""
}

func (b rawBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.data)
}

func toBlockSet(blocks []map[string]any) []slackapi.Block {
	blockSet := make([]slackapi.Block, len(blocks))
	for i, b := range blocks {
		blockSet[i] = rawBlock{data: b}
	}
	return blockSet
}
