import json
from pydantic import ValidationError
from connectors.grafana.src.telemetry.logging import log


class SettingsLoader:
    @staticmethod
    def load():
        from connectors.grafana.settings import Settings

        try:
            settings = Settings()

        except ValidationError as err:
            print(json.dumps([
                {
                    'error': error['msg'],
                    'env': f'{error["loc"][0].upper()}'
                } for error in err.errors()
            ], indent=2))
            raise SystemExit(1)
        
        return settings
    

class RoutesLoader:
    @staticmethod
    def load(app):
        from connectors.grafana.settings import settings
        from connectors.grafana.src.routes import internal

        app.include_router(internal.router, prefix="/api")

        log.info(f'Listening for incoming query requests on {settings.listen_host}:{settings.listen_port} and forwarding to Grafana at {settings.url}')
