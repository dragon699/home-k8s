package grafana_alert

import (
	"encoding/json"
	"fmt"

	"connector-slack/internal/config"
	"connector-slack/internal/notifications"
	"connector-slack/internal/slack"
	t "connector-slack/internal/telemetry"

	slackapi "github.com/slack-go/slack"
)

func ButtonInvestigate(value string, message slackapi.Message, user string) {
	var alert notifications.GrafanaAlertItem
	askMsg := []map[string]any{
		{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": "@Openclaw, check this one :point_up::skin-tone-3:",
			},
		},
	}

	alertJSON, err := json.Marshal(message.Metadata.EventPayload)

	if err == nil {
		_ = json.Unmarshal(alertJSON, &alert)
	}

	_, err = slack.Client.SendMsg(
		config.Config.SlackGrafanaAlertsChannelID,
		askMsg,
		nil,
		slackapi.MsgOptionIconURL(config.Config.SlackGrafanaIconURL),
		slackapi.MsgOptionUsername(config.Config.SlackGrafanaUsername),
		slackapi.MsgOptionText(fmt.Sprintf("%s > Investigation requested from Stitch", alert.Labels["alert_name"]), false),
		slackapi.MsgOptionTS(message.Timestamp),
	)

	if err != nil {
		t.Log.Error("Failed to send message", "error", err)
	}

	msgStatusUpdated := false

	for i := len(message.Blocks.BlockSet) - 1; i >= 0; i-- {
		actionsBlock, ok := message.Blocks.BlockSet[i].(*slackapi.ActionBlock)

		if !ok || len(actionsBlock.Elements.ElementSet) == 0 {
			continue
		}

		for _, element := range actionsBlock.Elements.ElementSet {
			button, ok := element.(*slackapi.ButtonBlockElement)

			if !ok || button.ActionID != "grafana_alert_button_investigate" {
				continue
			}

			remaining := append(message.Blocks.BlockSet[:i], message.Blocks.BlockSet[i+1:]...)

			var blocks []map[string]any
			if raw, err := json.Marshal(remaining); err == nil {
				_ = json.Unmarshal(raw, &blocks)
			}

			if _, err := slack.Client.UpdateMsg(config.Config.SlackGrafanaAlertsChannelID, message.Timestamp, blocks, nil); err != nil {
				t.Log.Error("Failed to update message status", "error", err)
				return
			}

			msgStatusUpdated = true
			break
		}

		if msgStatusUpdated {
			break
		}
	}
}
