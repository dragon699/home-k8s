package response

type Torrent struct {
	ID                 int64    `json:"id,omitempty"`
	Name               string   `json:"name"`
	Hash               string   `json:"hash,omitempty"`
	Category           string   `json:"category,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	Status             string   `json:"status,omitempty"`
	ProgressPercentage float64  `json:"progress_percentage,omitempty"`
	EtaMinutes         int64    `json:"eta_minutes,omitempty"`
	MagnetURI          string   `json:"magnet_uri,omitempty"`
	Leechers           int64    `json:"leechers,omitempty"`
	Seeders            int64    `json:"seeders,omitempty"`

	DateAdded        string `json:"date_added,omitempty"`
	DateLastActivity string `json:"date_last_activity,omitempty"`
	DateCompleted    string `json:"date_completed,omitempty"`

	SizeTotalMB      int64   `json:"size_total_mb,omitempty"`
	SizeTotalGB      float64 `json:"size_total_gb,omitempty"`
	SizeDownloadedMB int64   `json:"size_downloaded_mb,omitempty"`
	SizeUploadedMB   int64   `json:"size_uploaded_mb,omitempty"`
	SizeLeftMB       int64   `json:"size_left_mb,omitempty"`
	SizeMB           int64   `json:"size_mb,omitempty"`

	FilesCount                  int64   `json:"files_count,omitempty"`
	FilesAvailabilityPercentage float64 `json:"files_availability_percentage,omitempty"`
	FilesPath                   string  `json:"files_path,omitempty"`
	SavePath                    string  `json:"save_path,omitempty"`

	SpeedDownloadMBps float64 `json:"speed_download_mbps,omitempty"`
	SpeedUploadMBps   float64 `json:"speed_upload_mbps,omitempty"`

	IMDB string      `json:"imdb,omitempty"`
	Meta TorrentMeta `json:"meta"`
}

type TorrentMeta struct {
	ManagedBy        string                       `json:"managed_by,omitempty"`
	ScheduledActions []TorrentMetaScheduledAction `json:"scheduled_actions,omitempty"`
}

type TorrentMetaScheduledAction struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Category    string `json:"category,omitempty"`
}

type QBTorrent struct {
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

type TPBTorrent struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Leechers string `json:"leechers"`
	Seeders  string `json:"seeders"`
	Size     string `json:"size"`
	NumFiles string `json:"num_files"`
	Added    string `json:"added"`
	IMDB     string `json:"imdb,omitempty"`
}
