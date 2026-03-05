package grafana_alert

import (
	"encoding/json"
	"fmt"

	"common/utils"
	"connector-slack/internal/config"
	"connector-slack/internal/notifications"
	"connector-slack/internal/slack"
	t "connector-slack/internal/telemetry"

	slackapi "github.com/slack-go/slack"
)

func ButtonInvestigate(value string, message slackapi.Message, user string) {
	var alert notifications.GrafanaAlertItem
	req := utils.Req{}

	if value == "completed" {
		slack.Client.SendEphemeralMsg(
			config.Config.SlackGrafanaAlertsChannelID, user, nil, nil,
			slackapi.MsgOptionText("Summary already attached to this message.", false),
			slackapi.MsgOptionUsername(config.Config.SlackAIUsername),
			slackapi.MsgOptionIconURL(config.Config.SlackAIIconURL),
		)
		return
	}

	if value == "in_progress" {
		slack.Client.SendEphemeralMsg(
			config.Config.SlackGrafanaAlertsChannelID, user, nil, nil,
			slackapi.MsgOptionText("Hold your horses, I'm investigating.. Will reply to this message when ready.", false),
			slackapi.MsgOptionUsername(config.Config.SlackAIUsername),
			slackapi.MsgOptionIconURL(config.Config.SlackAIIconURL),
		)
		return
	}

	if value != "pending" {
		return
	}

	if raw, err := json.Marshal(message.Metadata.EventPayload); err == nil {
		_ = json.Unmarshal(raw, &alert)
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

			button.Text.Text = "Investigating.."
			button.Value = "in_progress"

			var blocks []map[string]any
			if raw, err := json.Marshal(message.Blocks.BlockSet); err == nil {
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

	templatePath := fmt.Sprintf("%s/notifications/grafana/%s.tpl", config.Config.TemplatesBasePath, "alert_summary")
	aiMessage, err := slack.Client.SendMsgFromTemplate(
		config.Config.SlackGrafanaAlertsChannelID, "ai", templatePath, alert,
		slackapi.MsgOptionTS(message.Timestamp),
	)

	if err != nil {
		t.Log.Error("Failed to send alert summary message", "error", err)
		return
	}

	aiBlocks := aiMessage.Blocks

	if _, err = req.GET(fmt.Sprintf("%s/api/health", config.Config.ConnectorMLURL), nil, nil); err != nil {
		t.Log.Error(fmt.Sprintf("Connection to connector-ml failed: %s", config.Config.ConnectorMLURL), "error", err)

		aiBlocks[0]["status"] = "error"
		aiBlocks[0]["output"] = map[string]any{
			"type": "rich_text",
			"elements": []any{
				map[string]any{
					"type": "rich_text_section",
					"elements": []any{
						map[string]any{
							"type": "text",
							"text": "Failed to ping ",
						},
						map[string]any{
							"type": "text",
							"text": "connector-ml",
							"style": map[string]any{
								"bold": true,
							},
						},
						map[string]any{
							"type": "text",
							"text": "!",
						},
					},
				},
			},
		}

		if _, err := slack.Client.UpdateMsg(config.Config.SlackGrafanaAlertsChannelID, aiMessage.Timestamp, aiBlocks, nil); err != nil {
			t.Log.Error("Failed to update message status", "error", err)
		}

		return
	}

	aiBlocks[0]["details"] = map[string]any{
		"type": "rich_text",
		"elements": []any{
			map[string]any{
				"type": "rich_text_section",
				"elements": []any{
					map[string]any{
						"type": "text",
						"text": "Pinged ",
					},
					map[string]any{
						"type": "text",
						"text": "connector-ml",
						"style": map[string]any{
							"bold": true,
						},
					},
					map[string]any{
						"type": "text",
						"text": ".",
					},
				},
			},
		},
	}
	aiBlocks[0]["output"] = map[string]any{
		"type": "rich_text",
		"elements": []any{
			map[string]any{
				"type": "rich_text_section",
				"elements": []any{
					map[string]any{
						"type": "text",
						"text": "Submitting alert metadata to ",
					},
					map[string]any{
						"type": "text",
						"text": "Ollama",
						"style": map[string]any{
							"bold": true,
						},
					},
					map[string]any{
						"type": "text",
						"text": "..",
					},
				},
			},
		},
	}

	aiMessage, err = slack.Client.UpdateMsg(config.Config.SlackGrafanaAlertsChannelID, aiMessage.Timestamp, aiBlocks, nil)

	if err != nil {
		t.Log.Error("Failed to update message status", "error", err)
		return
	}
}
