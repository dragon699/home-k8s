package qbittorrent

type Torrent struct {
	Name      string  `json:"name"`
	Hash      string  `json:"hash,omitempty"`
	Category  string  `json:"category,omitempty"`
	Tags      string  `json:"tags,omitempty"`
	State     string  `json:"state,omitempty"`
	Progress  float64 `json:"progress,omitempty"`
	ETA       float64 `json:"eta,omitempty"`
	MagnetURI string  `json:"magnet_uri,omitempty"`
	Leechers  float64 `json:"num_leechs,omitempty"`
	Seeders   float64 `json:"num_seeds,omitempty"`

	AddedOn      float64 `json:"added_on,omitempty"`
	LastActivity float64 `json:"last_activity,omitempty"`
	CompletionOn float64 `json:"completion_on,omitempty"`

	TotalSize  float64 `json:"total_size,omitempty"`
	Downloaded float64 `json:"downloaded,omitempty"`
	Uploaded   float64 `json:"uploaded,omitempty"`
	AmountLeft float64 `json:"amount_left,omitempty"`
	Size       float64 `json:"size,omitempty"`

	Availability float64 `json:"availability,omitempty"`
	ContentPath  string  `json:"content_path,omitempty"`
	SavePath     string  `json:"save_path,omitempty"`

	SpeedDownload float64 `json:"dlspeed,omitempty"`
	SpeedUpload   float64 `json:"upspeed,omitempty"`
}
