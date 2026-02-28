{
  "channel": "{{ .Channel }}",
  "text": "{{ if .Title }}{{ .Title }}{{ else if .Message }}{{ .Message }}{{ else }}Grafana alert: {{ .Status }}{{ end }}",
  "unfurl_links": false,
  "blocks": [
    {
      "type": "header",
      "text": {
        "type": "plain_text",
        "text": "{{ if .Title }}{{ .Title }}{{ else }}Grafana alert: {{ .Status }}{{ end }}"
      }
    },
    {
      "type": "section",
      "fields": [
        {
          "type": "mrkdwn",
          "text": "*Status*\n{{ if .Status }}{{ .Status }}{{ else }}unknown{{ end }}"
        },
        {
          "type": "mrkdwn",
          "text": "*Receiver*\n{{ if .Receiver }}{{ .Receiver }}{{ else }}n/a{{ end }}"
        }
      ]
    },
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "{{ if .Message }}{{ .Message }}{{ else }}{{ if .CommonAnnotations.summary }}{{ .CommonAnnotations.summary }}{{ else }}No summary provided.{{ end }}{{ end }}"
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
          "url": "{{ .ExternalURL }}"
        }
      ]
    }
    {{- end }}
  ]
}
