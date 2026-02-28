package request

type GrafanaSlackPayload struct {
	Receiver          string            `json:"receiver,omitempty"`
	Status            string            `json:"status,omitempty"`
	OrgID             int64             `json:"orgId,omitempty"`
	Title             string            `json:"title,omitempty"`
	Message           string            `json:"message,omitempty"`
	Channel           string            `json:"channel,omitempty"`
	Version           string            `json:"version,omitempty"`
	GroupKey          string            `json:"groupKey,omitempty"`
	TruncatedAlerts   int               `json:"truncatedAlerts,omitempty"`
	State             string            `json:"state,omitempty"`
	ExternalURL       string            `json:"externalURL,omitempty"`
	GroupLabels       map[string]string `json:"groupLabels,omitempty"`
	CommonLabels      map[string]string `json:"commonLabels,omitempty"`
	CommonAnnotations map[string]string `json:"commonAnnotations,omitempty"`
	Alerts            []GrafanaAlert    `json:"alerts,omitempty"`
}

type GrafanaAlert struct {
	Status       string            `json:"status,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	StartsAt     string            `json:"startsAt,omitempty"`
	EndsAt       string            `json:"endsAt,omitempty"`
	Values       map[string]any    `json:"values,omitempty"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
	Fingerprint  string            `json:"fingerprint,omitempty"`
	SilenceURL   string            `json:"silenceURL,omitempty"`
	DashboardURL string            `json:"dashboardURL,omitempty"`
	PanelURL     string            `json:"panelURL,omitempty"`
	ImageURL     string            `json:"imageURL,omitempty"`
	ValueString  string            `json:"valueString,omitempty"`
}

type DownloaderSlackPayload struct {
	Event    string         `json:"event"`
	Title    string         `json:"title,omitempty"`
	Message  string         `json:"message,omitempty"`
	Channel  string         `json:"channel,omitempty"`
	Severity string         `json:"severity,omitempty"`
	Hash     string         `json:"hash,omitempty"`
	Category string         `json:"category,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
	Fields   []SlackField   `json:"fields,omitempty"`
	Actions  []SlackAction  `json:"actions,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

type SlackAction struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label"`
	Style string `json:"style,omitempty"`
	Value string `json:"value,omitempty"`
	URL   string `json:"url,omitempty"`
}
