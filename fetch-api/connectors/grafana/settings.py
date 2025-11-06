from connectors.grafana.src.telemetry.logging import logger
from connectors.grafana.src.loaders import SettingsLoader
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    name: str = 'connector-grafana'
    listen_host: str = '0.0.0.0'
    listen_port: int = 8080

    url: str
    sa_token: str

    otel_service_name: str = 'connector-grafana'
    otel_service_namespace: str = 'fetch-api'
    otel_service_version: str = 'undefined'
    otlp_endpoint_grpc: str = 'grpc.k8s.iaminyourpc.xyz:80'

    log_level: str = 'info'
    log_format: str = 'json'

    health_endpoint: str | None = None
    health_job_id: str | None = None
    health_next_check: str | None = None
    health_last_check: str | None = None
    healthy: bool | None = None

    health_check_interval_seconds: int = 180
    health_retry_interval_seconds: int = 15

    def model_post_init(self, __context):
        logger.update_settings(
            log_level=self.log_level,
            log_format=self.log_format
        )

        if not self.health_endpoint:
            self.health_endpoint = f'{self.url}/api/health'


settings = SettingsLoader.load()
