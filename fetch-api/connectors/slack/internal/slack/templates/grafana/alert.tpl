{{- $slackChannel := "C0AJMC5RMLH" -}}
{{- $slackAsUser := "Grafana" -}}
{{- $slackAsUserIcon := "https://cdn.iconscout.com/icon/free/png-512/free-grafana-logo-icon-svg-download-png-2944910.png?f=webp&w=512" -}}

{{- $status := "Firing" -}}

{{- $titleIcon := ":grafana_alert:" -}}
{{- if eq .Status "resolved" -}}
  {{- $status = "Recovered" -}}
  {{- $titleIcon = ":grafana_alert_recovered:" -}}
{{- end -}}
{{- $titleText := or (index .Annotations "summary") (index .Labels "alertname") -}}

{{- $notification := printf "%s > %s" (toUpper $status) $titleText -}}
{{- $title := printf "%s *%s* > %s" $titleIcon $status $titleText -}}
{{- $description := or (index .Annotations "description") "" -}}

{{- $footer := printf "Detected %s" (beautifyTime .StartsAt) -}}
{{- if not (hasPrefix .EndsAt "0001") -}}
  {{- $footer = printf "%s\nRecovered %s" $footer (beautifyTime .EndsAt) -}}
{{- end -}}

{{- $dashboardURL := or .PanelURL .DashboardURL -}}
{{- $screenshotURL := .ImageURL -}}


{
  "channel": {{ json $slackChannel }},
  "username": {{ json $slackAsUser }},
  "icon_url": {{ json $slackAsUserIcon }},
  "text": {{ json $notification }},
  "blocks": [
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": {{ json $title }}
			}
		},
		{{- if $description -}}
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": {{ json $description }}
			}
		},
		{{- end -}}
		{
			"type": "section",
			"fields": [
				{
					"type": "mrkdwn",
					"text": {{ json (printf "*Status*\n%s" $status) }}
				}
				{{- range $key, $value := .Labels }}
				{{- if and (ne $key "alertname") (not (hasPrefix $key "grafana_")) }},
				{
					"type": "mrkdwn",
					"text": {{ json (printf "*%s*\n%s" (capitalize $key) $value) }}
				}
				{{- end }}
				{{- end }}
				{{- range $key, $value := .Annotations }}
				{{- if and (ne $key "summary") (ne $key "description") }},
				{
					"type": "mrkdwn",
					"text": {{ json (printf "*%s*\n%s" (capitalize $key) $value) }}
				}
				{{- end }}
				{{- end }}
			]
		},
		{
			"type": "context",
			"elements": [
				{
					"type": "mrkdwn",
					"text": {{ json $footer }}
				}
			]
		},
		{
			"type": "actions",
			"elements": [
				{
					"type": "button",
					"text": {
						"type": "plain_text",
						"text": "☉ Investigate"
					},
					"action_id": "grafana_alert_button_investigate"
				},
				{
					"type": "button",
					"text": {
						"type": "plain_text",
						"text": "⇲  Values"
					},
					"action_id": "grafana_alert_button_values"
				}
				{{- if $dashboardURL -}},
				{
					"type": "button",
					"text": {
						"type": "plain_text",
						"text": "⚎  View in dashboard"
					},
					"url": {{ json $dashboardURL }},
					"action_id": "grafana_alert_button_view_in_dashboard"
				}
				{{- end }}
			]
		}
		{{- if $screenshotURL -}},
		{
			"type": "image",
			"image_url": {{ json $screenshotURL }},
			"alt_text": "From the dashboard"
		}
	]
}
