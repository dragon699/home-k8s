package telemetry


import (
	"log/slog"
	"os"
	"strings"
)


func getLogger(levelName string, formatName string) *slog.Logger {
    level := slog.LevelInfo

    switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
		case "debug":
			level = slog.LevelDebug
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
    }

    format := strings.ToLower(os.Getenv("LOG_FORMAT"))
    var handler slog.Handler

    switch format {
		case "logfmt", "text", "":
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
		case "json":
			handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
		default:
			handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
    }

    return slog.New(handler)
}


var Logger = getLogger(os.Getenv("LOG_LEVEL"), os.Getenv("LOG_FORMAT"))
