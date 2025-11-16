import requests


DEFAULT_TIMEOUT = 10


def get(url: str, headers: dict = {}, params: dict = {}, json: dict = {}, timeout: int = DEFAULT_TIMEOUT):
    response = requests.get(
        url,
        headers=headers,
        params=params,
        json=json,
        timeout=timeout
    )

    return response


def post(url: str, headers: dict = {}, params: dict = {}, json: dict = {}, timeout: int = DEFAULT_TIMEOUT):
    response = requests.post(
        url,
        headers=headers,
        params=params,
        json=json,
        timeout=timeout
    )

    return response
