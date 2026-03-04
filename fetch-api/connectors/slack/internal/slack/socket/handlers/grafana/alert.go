package grafana

import (
	t "connector-slack/internal/telemetry"
)

func ButtonInvestigate(alertMeta string) {
	t.Log.Info("Debug", "alertMeta", alertMeta)
}
