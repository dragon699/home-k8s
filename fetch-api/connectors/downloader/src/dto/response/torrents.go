package response


type Torrent struct {
	AddedOn         int64   `json:"added_on"`
	Availability    float64 `json:"availability"`
	CompletionOn    int64   `json:"completion_on,omitempty"`
	ContentPath     string  `json:"content_path,omitempty"`
	DownloadSpeed   int64   `json:"download_speed,omitempty"`
	Downloaded      int64   `json:"downloaded,omitempty"`
}

type TorrentListResponse = BaseResponse[Torrent]
