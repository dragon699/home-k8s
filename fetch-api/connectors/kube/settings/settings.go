package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"common/utils"
)


type Settings struct {
	Name       string                 `json:"name"                          env:"NAME"`
	ListenHost string                 `json:"listen_host"                   env:"LISTEN_HOST"`
	ListenPort int                    `json:"listen_port"                   env:"LISTEN_PORT"`

	InCluster      bool               `json:"in_cluster"                    env:"IN_CLUSTER"`
	KubeConfigPath string             `json:"kube_config_path"              env:"KUBE_CONFIG_PATH"`

	OtelServiceName      string       `json:"otel_service_name"             env:"OTEL_SERVICE_NAME"`
	OtelServiceNamespace string       `json:"otel_service_namespace"        env:"OTEL_SERVICE_NAMESPACE"`
	OtelServiceVersion   string       `json:"otel_service_version"          env:"OTEL_SERVICE_VERSION"`
	OtlpEndpointGrpc     string       `json:"otlp_endpoint_grpc"            env:"OTLP_ENDPOINT_GRPC"`

	LogLevel  string                  `json:"log_level"                     env:"LOG_LEVEL"`
	LogFormat string                  `json:"log_format"                    env:"LOG_FORMAT"`

	HealthCheckIntervalSeconds int    `json:"health_check_interval_seconds" env:"HEALTH_CHECK_INTERVAL_SECONDS"`
	HealthRetryIntervalSeconds int    `json:"health_retry_interval_seconds" env:"HEALTH_RETRY_INTERVAL_SECONDS"`
	HealthEndpoint             string `json:"health_endpoint"               env:"HEALTH_ENDPOINT"`
	Healthy                    *bool  `json:"healthy"`
}

var defaultSettings = Settings{
	Name:                             "connector-kube",
	ListenHost:                       "0.0.0.0",
	ListenPort:             		  8080,

	InCluster:                        true,
	KubeConfigPath:					  "",

	OtelServiceName:                  "connector-kube",
	OtelServiceNamespace:             "fetch-api",
	OtelServiceVersion:               "",
	OtlpEndpointGrpc:                 "grafana-alloy.monitoring.svc:4317",

	LogLevel:                         "info",
	LogFormat:                        "json",

	HealthCheckIntervalSeconds:       180,
	HealthRetryIntervalSeconds:       15,
	HealthEndpoint:                   "/healthz",
	Healthy:                          nil,
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

	if !settings.InCluster && settings.KubeConfigPath == "" {
		return Settings{}, fmt.Errorf("KUBE_CONFIG_PATH is required when IN_CLUSTER is set to false")
	}

	if settings.OtelServiceVersion == "" {
		currentDir, err := utils.GetCurrentDir()

		if err != nil {
			settings.OtelServiceVersion = "unknown"
		} else {
			verFile := filepath.Join(currentDir, "..", "VERSION")

			if appVer, err := utils.ReadFile(verFile); err != nil {
				settings.OtelServiceVersion = "unknown"
			} else {
				settings.OtelServiceVersion = strings.TrimSpace(appVer)
			}
		}
	}

    return settings, nil
}
