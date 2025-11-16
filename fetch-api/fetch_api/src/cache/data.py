from common.telemetry.src.tracing.wrappers import traced



class CachedResponse:
    def __init__(self, cached_at: str = '', status_code: int = 200, headers: dict = {}, json_data: dict = {}):
        self.cached_at = cached_at
        self.status_code = status_code
        self.headers = headers
        self._json = json_data

    @traced()
    def json(self, span=None):
        return {
            'cached': True,
            'cached_at': self.cached_at,
            **self._json
        }
