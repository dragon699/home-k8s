from fetch_api.settings import connectors
from common.utils.web import create_session
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword



class ConnectorClient:
    def __init__(self, connector_name: str):
        self.connector_name = connector_name
        self.url = connectors[connector_name].url
        self.set_headers()


    def set_headers(self):
        self.headers = {
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        }


    @staticmethod
    @traced()
    def ping(connector_name: str, connector_url: str, health_endpoint: str, span=None):
        span.set_attributes({
            'connector.name': connector_name,
            'connector.url': connector_url,
            'connector.endpoint': health_endpoint,
            'connector.operation': 'ping'
        })

        session = create_session(timeout=5)
        response = session.get(health_endpoint)

        return response


    @traced()
    def get(self, endpoint: str, params: dict = {}, data: dict = {}, span=None):
        span.set_attributes(
            reword({
                'connector.name': self.connector_name,
                'connector.method': 'GET',
                'connector.request.params': params,
                'connector.request.body': data,
                'connector.url': self.url,
                'connector.endpoint': endpoint
            })
        )

        try:
            session = create_session(timeout=5)
            response = session.get(
                f'{self.url}/{endpoint}',
                headers=self.headers,
                params=params,
                json=data
            )

            return response

        except Exception as err:
            span.set_attributes(
                reword({
                    'connector.error.message': str(err),
                    'connector.error.type': type(err).__name__
                })
            )


    @traced()
    def post(self, endpoint: str, params: dict = {}, data: dict = {}, span=None):
        span.set_attributes(
            reword({
                'connector.name': self.connector_name,
                'connector.method': 'POST',
                'connector.request.params': params,
                'connector.request.body': data,
                'connector.url': self.url,
                'connector.endpoint': endpoint
            })
        )

        try:
            session = create_session(timeout=5)
            response = session.post(
                f'{self.url}/{endpoint}',
                headers=self.headers,
                params=params,
                json=data
            )

            return response

        except Exception as err:
            span.set_attributes(
                reword({
                    'connector.error.message': str(err),
                    'connector.error.type': type(err).__name__
                })
            )
