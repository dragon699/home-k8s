import requests
from requests import exceptions as ReqExceptions
from fetch_api.src.cache.client import RedisClient
from fetch_api.src.cache.data import CachedResponse
from fetch_api.settings import settings, connectors
from fetch_api.src.telemetry.logging import log
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword
from common.utils.helpers import time_now, create_cache_key



class ConnectorClient:
    def __init__(self, connector_name: str, requests_timeout: int = 5, cache: bool = False):
        self.connector_name = connector_name
        self.url = connectors[connector_name].url
        self.requests_timeout = requests_timeout
        self.cache = cache
        self.set_headers()

        if self.cache:
            if any(v is None for v in [
                settings.redis_host,
                settings.redis_port,
                settings.redis_db
            ]):
                log.critical(
                    f'{self.connector_name} requires caching configuration, but Redis settings are incomplete',
                    connector=self.connector_name
                )
                raise SystemExit(1)

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

        response = requests.get(
            health_endpoint,
            timeout=5
        )

        return response


    @traced()
    def get(self, endpoint: str, params: dict = {}, data: dict = {}, cache_key: tuple | None = None, span=None):
        span.set_attributes(
            reword({
                'connector.name': self.connector_name,
                'connector.method': 'GET',
                'connector.request.params': params,
                'connector.request.body': data,
                'connector.url': self.url,
                'connector.endpoint': endpoint,
                'connector.cache.enabled': self.cache
            })
        )

        cached_value = None

        if self.cache:
            if cache_key is None:
                cache_key = create_cache_key(
                    connector_name=self.connector_name,
                    method='GET',
                    endpoint=endpoint,
                    params=params,
                    data=data
                )

            cached_value = self.redis.get(cache_key)

        try:
            response = requests.get(
                f'{self.url}/{endpoint}',
                headers=self.headers,
                params=params,
                json=data,
                timeout=self.requests_timeout
            )

            if response.status_code in [200, 201]:
                if self.cache:
                    self.redis.set(
                        cache_key,
                        {
                            'cached_at': time_now(),
                            'status_code': response.status_code,
                            'json': response.json()
                        },
                        ttl=settings.redis_cache_ttl
                    )

                    span.set_attributes({
                        'connector.cache.updated': True,
                        'connector.cache.updated_at': time_now(),
                        'connector.cache.key': cache_key
                    })

            else:
                if self.cache and not cached_value is None:
                    span.set_attributes({
                        'connector.cache.status': 'hit',
                        'connector.cache.updated': False,
                        'connector.cache.key': cache_key,
                        'connector.cache.cached_at': cached_value['cached_at']
                    })

                    return CachedResponse(
                        cached_at=cached_value['cached_at'],
                        status_code=cached_value['status_code'],
                        json_data=cached_value['json']
                    )
                
            return response

        except ReqExceptions.RequestException as err:
            if self.cache and not cached_value is None:
                span.set_attributes({
                    'connector.cache.status': 'hit',
                    'connector.cache.updated': False,
                    'connector.cache.key': cache_key,
                    'connector.cache.cached_at': cached_value['cached_at']
                })

                return CachedResponse(
                    cached_at=cached_value['cached_at'],
                    status_code=cached_value['status_code'],
                    json_data=cached_value['json']
                )

            span.set_attributes({
                'connector.cache.enabled': self.cache,
                'connector.error.message': str(err),
                'connector.error.type': type(err).__name__
            })

            raise err


    @traced()
    def post(self, endpoint: str, params: dict = {}, data: dict = {}, cache_key: tuple | None = None, span=None):
        span.set_attributes(
            reword({
                'connector.name': self.connector_name,
                'connector.method': 'POST',
                'connector.request.params': params,
                'connector.request.body': data,
                'connector.url': self.url,
                'connector.endpoint': endpoint,
                'connector.cache.enabled': self.cache
            })
        )

        cached_value = None

        if self.cache:
            if cache_key is None:
                cache_key = create_cache_key(
                    connector_name=self.connector_name,
                    method='POST',
                    endpoint=endpoint,
                    params=params,
                    data=data
                )

            cached_value = self.redis.get(cache_key)

        try:
            response = requests.post(
                f'{self.url}/{endpoint}',
                headers=self.headers,
                params=params,
                json=data,
                timeout=self.requests_timeout
            )

            if response.status_code in [200, 201]:
                if self.cache:
                    self.redis.set(
                        cache_key,
                        {
                            'cached_at': time_now(),
                            'status_code': response.status_code,
                            'json': response.json()
                        },
                        ttl=settings.redis_cache_ttl
                    )

                    span.set_attributes({
                        'connector.cache.updated': True,
                        'connector.cache.updated_at': time_now(),
                        'connector.cache.key': cache_key
                    })

            else:
                if self.cache and not cached_value is None:
                    span.set_attributes({
                        'connector.cache.status': 'hit',
                        'connector.cache.updated': False,
                        'connector.cache.key': cache_key,
                        'connector.cache.cached_at': cached_value['cached_at']
                    })

                    return CachedResponse(
                        cached_at=cached_value['cached_at'],
                        status_code=cached_value['status_code'],
                        json_data=cached_value['json']
                    )
                
            return response

        except ReqExceptions.RequestException as err:
            if self.cache and not cached_value is None:
                span.set_attributes({
                    'connector.cache.status': 'hit',
                    'connector.cache.updated': False,
                    'connector.cache.key': cache_key,
                    'connector.cache.cached_at': cached_value['cached_at']
                })

                return CachedResponse(
                    cached_at=cached_value['cached_at'],
                    status_code=cached_value['status_code'],
                    json_data=cached_value['json']
                )

            span.set_attributes({
                'connector.cache.enabled': self.cache,
                'connector.error.message': str(err),
                'connector.error.type': type(err).__name__
            })

            raise err
