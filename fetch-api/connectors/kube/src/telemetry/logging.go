package telemetry

import (
	"log/slog"

	"common/telemetry"
	"connector-kube/settings"
)

var Log *slog.Logger


func init() {
	Log = telemetry.NewLogger(
		settings.Config.LogLevel,
		settings.Config.LogFormat,
	)
}
