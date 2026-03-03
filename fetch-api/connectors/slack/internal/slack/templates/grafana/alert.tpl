{
  "channel": {{ json .Channel }},
  "username": "Alerts",
  "icon_url": "https://cdn.iconscout.com/icon/free/png-512/free-grafana-logo-icon-svg-download-png-2944910.png?f=webp&w=512",
  "text": {{ json (or .Title (or .Message (printf "Grafana alert: %s" (or .Status "unknown")))) }},
  "blocks": [
    {
      "type": "header",
      "text": {
        "type": "plain_text",
        "text": {{ json (or .Title (printf "Grafana alert: %s" (or .Status "unknown"))) }}
      }
    },
    {
      "type": "section",
      "fields": [
        {
          "type": "mrkdwn",
          "text": {{ json (printf "*Status*\n%s" (or .Status "unknown")) }}
        },
        {
          "type": "mrkdwn",
          "text": {{ json (printf "*Receiver*\n%s" (or .Receiver "n/a")) }}
        }
      ]
    },
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": {{ json (or .Message (or .CommonAnnotations.summary "No summary provided.")) }}
      }
    }
    {{- if .ExternalURL }},
    {
      "type": "actions",
      "elements": [
        {
          "type": "button",
          "text": {
          "type": "plain_text",
          "text": "Open Grafana"
          },
          "url": {{ json .ExternalURL }}
        }
      ]
    }
    {{- end }}
  ]
}
