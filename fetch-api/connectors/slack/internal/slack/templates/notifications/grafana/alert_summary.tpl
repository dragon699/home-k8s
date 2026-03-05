{
    "text": "Investigation summary is ready",
	"blocks": [
		{
			"type": "context",
			"elements": [
				{
					"type": "mrkdwn",
					"text": "Thinked *{{ .DurationSeconds }}* seconds."
				}
			]
		},
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "{{ json .Answer }}"
			}
		}
	]
}
