package slack

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
