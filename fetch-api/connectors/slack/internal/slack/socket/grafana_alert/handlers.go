package grafana_alert

import (
	"encoding/json"
	"fmt"
	"time"

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
			slackapi.MsgOptionText("Investigation summary already attached to this message.", false),
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

	alertJSON, err := json.Marshal(message.Metadata.EventPayload)

	if err == nil {
		_ = json.Unmarshal(alertJSON, &alert)
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

	templatePath := fmt.Sprintf("%s/notifications/grafana/%s.tpl", config.Config.TemplatesBasePath, "alert_summary_in_progress")
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
						"text": "I'm investigating..",
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

	var aiReqResponse *utils.Response
	var aiResponse ConnectorMLResponse
	var retries int = 0

	for {
		if retries == 60 {
			t.Log.Error("connector-ml failed to respond on time, so I gave up")

			aiBlocks[0]["title"] = "Investigation failed"
			aiBlocks[0]["status"] = "error"
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
								"text": "Ollama ",
								"style": map[string]any{
									"bold": true,
								},
							},
							map[string]any{
								"type": "text",
								"text": "didn't respond on time!",
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

		aiReqResponse, err = req.POST(
			fmt.Sprintf("%s/ask", config.Config.ConnectorMLURL),
			map[string]string{"Content-Type": "application/json"},
			nil,
			map[string]any{
				"prompt":                fmt.Sprintf("Now, analyze the following alert:\n%s", alertJSON),
				"instructions_template": "grafana-alerts",
			},
		)

		retries += 1

		if err == nil {
			break
		}

		t.Log.Error("connector-ml returned invalid response, or failed to respond on time, retrying in 5 seconds..", "error", err)
		time.Sleep(5 * time.Second)
	}

	aiBlocks[0]["title"] = "Summarizing results"
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
			map[string]any{
				"type": "rich_text_section",
				"elements": []any{
					map[string]any{
						"type": "text",
						"text": "Got my results from ",
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
						"text": "Summarizing what ",
					},
					map[string]any{
						"type": "text",
						"text": "Ollama ",
						"style": map[string]any{
							"bold": true,
						},
					},
					map[string]any{
						"type": "text",
						"text": "told me..",
					},
				},
			},
		},
	}

	aiMessage, err = slack.Client.UpdateMsg(config.Config.SlackGrafanaAlertsChannelID, aiMessage.Timestamp, aiBlocks, nil)
	time.Sleep(7 * time.Second)

	if err != nil {
		t.Log.Error("Failed to update message status", "error", err)
		return
	}

	if raw, err := json.Marshal(aiReqResponse.Body); err != nil {
		t.Log.Error("Failed to serialize connector-ml response", "error", err)
		return
	} else if err := json.Unmarshal(raw, &aiResponse); err != nil {
		t.Log.Error("Failed to parse connector-ml response", "error", err)
		return
	}

	if len(aiResponse.Items) == 0 {
		t.Log.Error("connector-ml response contained no items")
		return
	}

	msgStatusUpdated = false

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

			button.Text.Text = "Investigation attached"
			button.Value = "completed"
			button.Style = "primary"

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

	aiBlocks[0]["status"] = "complete"
	aiBlocks[0]["title"] = "Summary attached"
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
			map[string]any{
				"type": "rich_text_section",
				"elements": []any{
					map[string]any{
						"type": "text",
						"text": "Got my results from ",
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
						"text": ".",
					},
				},
			},
			map[string]any{
				"type": "rich_text_section",
				"elements": []any{
					map[string]any{
						"type": "text",
						"text": "Nice and tidy!",
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
						"text": "Summary attached",
						"style": map[string]any{
							"bold": true,
						},
					},
				},
			},
		},
	}

	if _, err := slack.Client.UpdateMsg(config.Config.SlackGrafanaAlertsChannelID, aiMessage.Timestamp, aiBlocks, nil); err != nil {
		t.Log.Error("Failed to update message status", "error", err)
	}

	templatePath = fmt.Sprintf("%s/notifications/grafana/%s.tpl", config.Config.TemplatesBasePath, "alert_summary")
	_, err = slack.Client.SendMsgFromTemplate(
		config.Config.SlackGrafanaAlertsChannelID, "ai", templatePath, aiResponse.Items[0],
		slackapi.MsgOptionTS(message.Timestamp),
	)

	if err != nil {
		t.Log.Error("Failed to send alert summary message", "error", err)
		return
	}
}
