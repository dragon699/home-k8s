from pydantic_settings import BaseSettings
from fetch_api.src.telemetry.logging import logger
from fetch_api.src.loaders import SettingsLoader



class FetchAPISettings(BaseSettings):
    name: str = 'fetch-api'
    listen_host: str = '0.0.0.0'
    listen_port: int = 8079

    otel_service_name: str = 'fetch-api'
    otel_service_namespace: str = 'fetch-api'
    otel_service_version: str = 'undefined'
    otlp_endpoint_grpc: str = 'grpc.k8s.iaminyourpc.xyz:80'

    log_level: str = 'info'
    log_format: str = 'json'

    connectors: str

    connector_health_check_interval_seconds: int = 20
    connector_health_retry_interval_seconds: int = 5


    def model_post_init(self, __context):
        logger.update_settings(
            log_level=self.log_level,
            log_format=self.log_format
        )


class ConnectorSettings(BaseSettings):
    name: str
    host: str
    port: int
    protocol: str = 'http'
    url: str | None = None

    health_endpoint: str | None = None
    health_job_id: str | None = None
    health_next_check: str | None = None
    health_last_check: str | None = None
    healthy: bool | None = None


    def model_post_init(self, __context):
        if not self.url:
            self.url = f'{self.protocol}://{self.host}:{self.port}'

        if not self.health_endpoint:
            self.health_endpoint = f'{self.url}/api/health'


settings, connectors = SettingsLoader().load()

