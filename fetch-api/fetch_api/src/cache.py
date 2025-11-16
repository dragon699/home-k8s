import json, redis
from common.telemetry.src.tracing.wrappers import traced



class RedisClient:
    def __init__(self, host: str, port: int, db: int, password: str | None = None):
        self.client = redis.Redis(
            host=host,
            port=port,
            db=db,
            password=password,
            decode_responses=True
        )


    @traced()
    def get(self, key: str, span=None):
        value = self.client.get(key)

        if value is None:
            return None

        try:
            return json.loads(value)

        except:
            return value


    @traced()
    def set(self, key: str, value, ttl: int | None = None, span=None):
        encoded_value = json.dumps(value)

        if ttl:
            self.client.setex(key, ttl, encoded_value)
        
        else:
            self.client.set(key, encoded_value)


class CachedResponse:
    def __init__(self, json_data, status_code=200, headers=None):
        self._json = json_data
        self.status_code = status_code
        self.headers = headers or {}

    def json(self):
        return self._json
