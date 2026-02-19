package request


type AddTorrentPayload struct {
	URL       string    `json:"url"`
	Category  string    `json:"category,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	SavePath  string    `json:"save_path,omitempty"`
	Manage    *bool     `json:"manage,omitempty"`
	Notify    *bool     `json:"notify,omitempty"`
}
