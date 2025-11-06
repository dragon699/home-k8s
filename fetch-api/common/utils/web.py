import requests
from requests.adapters import HTTPAdapter


TIMEOUT = 10

class Request(HTTPAdapter):
    def __init__(self, *args, **kwargs):
        self.timeout = kwargs.pop('timeout', TIMEOUT)

        super().__init__(*args, **kwargs)

    def send(self, request, **kwargs):
        kwargs.setdefault('timeout', self.timeout)

        return super().send(request, **kwargs)


def create_session(timeout: int = TIMEOUT):
    session = requests.Session()
    adapter = Request(timeout=timeout)

    session.mount('http://', adapter)
    session.mount('https://', adapter)

    return session
