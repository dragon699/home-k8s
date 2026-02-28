package response

type SlackDeliveryResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}
