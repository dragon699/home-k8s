from requests.exceptions import Timeout
from fetch_api.src.cache import RedisClient, CachedResponse
from fetch_api.settings import settings, connectors
from common.utils.web import create_session
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword



class ConnectorClient:
    def __init__(self, connector_name: str, requests_timeout: int = 5, cache: bool = False):
        self.connector_name = connector_name
        self.url = connectors[connector_name].url
        self.requests_timeout = requests_timeout
        self.cache = cache
        self.set_headers()

        if self.cache:
            self.redis = RedisClient(
                host=settings.redis_host,
                port=settings.redis_port,
                db=settings.redis_db,
                password=settings.redis_password
            )


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

        if self.cache:
            cache_key = (
                f'connector:{self.connector_name}:GET:{endpoint}:'
                f'{hash(frozenset(params.items()))}:'
                f'{hash(frozenset(data.items()))}'
            )
            cached_value = self.redis.get(cache_key)

            if not cached_value is None:
                return CachedResponse(
                    json_data=cached_value['json'],
                    status_code=cached_value['status_code']
                )

        try:
            session = create_session(timeout=self.requests_timeout)
            response = session.get(
                f'{self.url}/{endpoint}',
                headers=self.headers,
                params=params,
                json=data
            )

            if self.cache and response.status_code in [200, 201]:
                self.redis.set(
                    cache_key,
                    {
                        'json': response.json(),
                        'status_code': response.status_code
                    },
                    ttl=settings.redis_cache_ttl
                )

            return response
        
        except Timeout as err:
            if self.cache and not cached_value is None:
                print('CACHE HIT CUZ OF TIMEOUT')
                return CachedResponse(
                    json_data=cached_value['json'],
                    status_code=cached_value['status_code']
                )

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

        if self.cache:
            cache_key = (
                f'connector:{self.connector_name}:POST:{endpoint}:'
                f'{hash(frozenset(params.items()))}:'
                f'{hash(frozenset(data.items()))}'
            )
            cached_value = self.redis.get(cache_key)

            if not cached_value is None:
                return CachedResponse(
                    json_data=cached_value['json'],
                    status_code=cached_value['status_code']
                )

        try:
            session = create_session(timeout=self.requests_timeout)
            response = session.post(
                f'{self.url}/{endpoint}',
                headers=self.headers,
                params=params,
                json=data
            )

            if self.cache and response.status_code in [200, 201]:
                self.redis.set(
                    cache_key,
                    {
                        'json': response.json(),
                        'status_code': response.status_code
                    },
                    ttl=settings.redis_cache_ttl
                )

            return response
        
        except Timeout as err:
            if self.cache and not cached_value is None:
                return CachedResponse(
                    json_data=cached_value['json'],
                    status_code=cached_value['status_code']
                )

        except Exception as err:
            span.set_attributes(
                reword({
                    'connector.error.message': str(err),
                    'connector.error.type': type(err).__name__
                })
            )
