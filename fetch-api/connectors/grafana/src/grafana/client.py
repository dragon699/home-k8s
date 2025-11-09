from connectors.grafana.settings import settings
from common.utils.web import create_session
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword
from connectors.grafana.src.telemetry.logging import log



class GrafanaClient:
    def __init__(self):
        self.url = settings.url
        self.sa_token = settings.sa_token
        self.set_headers()


    def set_headers(self):
        self.headers = {
            'Authorization': f'Bearer {self.sa_token}',
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        }


    @staticmethod
    @traced()
    def ping(span=None):
        span.set_attributes({
            'grafana.operation': 'ping',
            'grafana.url': settings.url,
            'grafana.endpoint': settings.health_endpoint
        })

        session = create_session(timeout=5)
        response = session.get(settings.health_endpoint)
        
        return response


    @traced()
    def authenticate(self, span=None):
        if not settings.healthy:
            settings.authenticated = False
            return False

        try:
            session = create_session(timeout=5)
            response = session.get(
                settings.auth_endpoint,
                headers=self.headers
            )

            if response.status_code == 200:
                if not settings.authenticated:
                    log.debug(f'Authentication successful', extra={
                        'auth_endpoint': settings.auth_endpoint
                    })

                    settings.authenticated = True

                span.set_attributes(
                    reword({
                        'grafana.operation': 'authentication',
                        'grafana.method': 'GET',
                        'grafana.response.status_code': response.status_code,
                        'grafana.url': settings.url,
                        'grafana.endpoint': settings.auth_endpoint,
                        'grafana.auth.status': settings.authenticated
                    })
                )

                return True

            else:
                settings.authenticated = False

                log.critical(f'Authentication failed', extra={
                    'auth_endpoint': settings.auth_endpoint,
                    'response_status_code': response.status_code
                })
                span.set_attributes(
                    reword({
                        'grafana.operation': 'authentication',
                        'grafana.method': 'GET',
                        'grafana.response.content': response.text,
                        'grafana.response.status_code': response.status_code,
                        'grafana.url': settings.url,
                        'grafana.endpoint': settings.auth_endpoint,
                        'grafana.auth.status': settings.authenticated
                    })
                )

                return False

        except Exception as err:
            settings.authenticated = False

            log.critical(f'Authentication failed, Grafana is unreachable', extra={
                'auth_endpoint': settings.auth_endpoint,
                'error': str(err)
            })
            span.set_attributes(
                reword({
                    'grafana.operation': 'authentication',
                    'grafana.method': 'GET',
                    'grafana.error.message': str(err),
                    'grafana.error.type': type(err).__name__,
                    'grafana.url': settings.url,
                    'grafana.endpoint': settings.auth_endpoint,
                    'grafana.auth.status': settings.authenticated
                })
            )

            return False


    @traced()
    def get(self, endpoint: str, data: dict = {}, span=None):
        if self.authenticate():
            span.set_attributes(
                reword({
                    'grafana.method': 'GET',
                    'grafana.request.body': data,
                    'grafana.url': settings.url,
                    'grafana.endpoint': endpoint,
                    'grafana.auth.status': settings.authenticated
                })
            )

            session = create_session(timeout=5)
            response = session.get(
                f'{self.url}/{endpoint}',
                headers=self.headers,
                json=data
            )

            return response
        
        else:
            span.set_attributes(
                reword({
                    'grafana.auth.status': settings.authenticated
                })
            )


    @traced()
    def post(self, endpoint: str, data: dict = {}, span=None):
        if self.authenticate():
            span.set_attributes(
                reword({
                    'grafana.method': 'POST',
                    'grafana.request.body': data,
                    'grafana.url': settings.url,
                    'grafana.endpoint': endpoint,
                    'grafana.auth.status': settings.authenticated
                })
            )

            session = create_session(timeout=5)
            response = session.post(
                f'{self.url}/{endpoint}',
                headers=self.headers,
                json=data
            )

            return response

        else:
            span.set_attributes(
                reword({
                    'grafana.auth.status': settings.authenticated
                })
            )
