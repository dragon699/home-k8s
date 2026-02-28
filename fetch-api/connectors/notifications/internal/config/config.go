package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"common/utils"
)

type Settings struct {
	Name       string `json:"connector_name"                env:"NAME"`
	ListenHost string `json:"listen_host"                   env:"LISTEN_HOST"`
	ListenPort int    `json:"listen_port"                   env:"LISTEN_PORT"`
	ListenUrl  string `json:"listen_url"`

	SlackBotToken          string `json:"slack_bot_token"           env:"SLACK_BOT_TOKEN"`
	SlackAppToken          string `json:"slack_app_token"           env:"SLACK_APP_TOKEN"`
	SlackDefaultChannel    string `json:"slack_default_channel"     env:"SLACK_DEFAULT_CHANNEL"`
	SlackGrafanaChannel    string `json:"slack_grafana_channel"     env:"SLACK_GRAFANA_CHANNEL"`
	SlackDownloaderChannel string `json:"slack_downloader_channel"  env:"SLACK_DOWNLOADER_CHANNEL"`
	SlackSocketModeEnabled bool   `json:"slack_socket_mode_enabled" env:"SLACK_SOCKET_MODE_ENABLED"`
	SlackSocketDebug       bool   `json:"slack_socket_debug"        env:"SLACK_SOCKET_DEBUG"`

	OtelServiceName      string `json:"otel_service_name"             env:"OTEL_SERVICE_NAME"`
	OtelServiceNamespace string `json:"otel_service_namespace"        env:"OTEL_SERVICE_NAMESPACE"`
	OtelServiceVersion   string `json:"otel_service_version"          env:"OTEL_SERVICE_VERSION"`
	OtlpEndpointGrpc     string `json:"otlp_endpoint_grpc"            env:"OTLP_ENDPOINT_GRPC"`

	LogLevel  string `json:"log_level"                     env:"LOG_LEVEL"`
	LogFormat string `json:"log_format"                    env:"LOG_FORMAT"`

	HealthCheckIntervalSeconds int `json:"health_check_interval_seconds" env:"HEALTH_CHECK_INTERVAL_SECONDS"`
	HealthRetryIntervalSeconds int `json:"health_retry_interval_seconds" env:"HEALTH_RETRY_INTERVAL_SECONDS"`
	Healthy                    *bool
	HealthJobID                *string
	HealthNextCheck            *string
	HealthLastCheck            *string
}

var defaultSettings = Settings{
	Name:       "notifications-controller",
	ListenHost: "0.0.0.0",
	ListenPort: 8080,

	SlackSocketModeEnabled: true,

	OtelServiceName:      "notifications-controller",
	OtelServiceNamespace: "fetch-api",
	OtelServiceVersion:   "",
	OtlpEndpointGrpc:     "grpc.k8s.iaminyourpc.xyz:80",

	LogLevel:  "info",
	LogFormat: "logfmt",

	HealthCheckIntervalSeconds: 15,
	HealthRetryIntervalSeconds: 5,
	Healthy:                    nil,
	HealthJobID:                nil,
}

var Config Settings

func init() {
	config, err := loadSettings()

	if err != nil {
		panic(err)
	}

	Config = config
}

func loadSettings() (Settings, error) {
	settings := Settings{}
	defaults := reflect.ValueOf(defaultSettings)

	fields := reflect.ValueOf(&settings).Elem()
	fieldsMeta := fields.Type()

	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		fieldMeta := fieldsMeta.Field(i)

		envName := fieldMeta.Tag.Get("env")
		if envName == "" {
			continue
		}

		envVal, envDefined := os.LookupEnv(envName)
		if !envDefined {
			field.Set(defaults.Field(i))

			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(envVal)

		case reflect.Int, reflect.Int64:
			val, err := utils.ToInt(envVal)

			if err != nil {
				return Settings{}, fmt.Errorf("%s got an incorrect value: %s", envName, err)
			}

			field.SetInt(val)

		case reflect.Bool:
			val, err := utils.ToBool(envVal)

			if err != nil {
				return Settings{}, fmt.Errorf("%s got an incorrect value: %s", envName, err)
			}

			field.SetBool(val)
		}
	}

	settings.ListenUrl = fmt.Sprintf("http://%s:%d", settings.ListenHost, settings.ListenPort)

	if settings.OtelServiceVersion == "" {
		_, currentFile, _, ok := runtime.Caller(0)

		if !ok {
			settings.OtelServiceVersion = "unknown"
		} else {
			currentDir := filepath.Dir(currentFile)
			verFile := filepath.Join(currentDir, "..", "..", "VERSION")

			if appVer, err := utils.ReadFile(verFile); err != nil {
				settings.OtelServiceVersion = "unknown"
			} else {
				settings.OtelServiceVersion = strings.TrimSpace(appVer)
			}
		}
	}

	return settings, nil
}
