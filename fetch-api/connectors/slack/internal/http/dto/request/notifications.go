package request

import (
	"connector-slack/internal/notifications"
)

type NotificationPayload struct {
	ChannelID string                     `json:"channel_id"`
	Blocks    []map[string]any           `json:"blocks,omitempty"`
	Options   NotificationOptionsPayload `json:"options"`
}

type NotificationOptionsPayload struct {
	User      string `json:"user,omitempty"`
	UserIcon  string `json:"user_icon,omitempty"`
	ExtraText string `json:"extra_text,omitempty"`
}

type TemplatedNotificationPayload[TemplateParams any] struct {
	NotificationName string         `json:"notification_name"`
	Params           TemplateParams `json:"params"`
}

type GrafanaAlertNotificationPayload = TemplatedNotificationPayload[notifications.GrafanaAlert]
type TorrentNotificationPayload = TemplatedNotificationPayload[notifications.Torrent]
