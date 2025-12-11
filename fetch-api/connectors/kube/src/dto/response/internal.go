package response


type Health struct {
	ConnectorName   string `json:"connector_name"`
	Healthy         *bool  `json:"healthy"`
	HealthLastCheck string `json:"health_last_check"`
	HealthNextCheck string `json:"health_next_check"`
}

type Ready struct {
	Ready           *bool  `json:"ready"`
}
