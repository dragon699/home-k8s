import json, urllib.request
from os import getenv


URL = f'http://localhost:{getenv("LISTEN_PORT")}'
ENDPOINT = 'api/healthz'

HEALTHZ_ENDPOINT = f'{URL}/{ENDPOINT}'


def is_up():
    return json.loads(
        urllib.request.urlopen(
            HEALTHZ_ENDPOINT
        ).read()
    )

def is_healthy(json_response):
    return json_response['healthy'] == True


try:
    status = is_up()

except:
    print(f'{HEALTHZ_ENDPOINT}: Unreachable!')
    raise SystemExit(1)

try:
    assert is_healthy(status)

except:
    print(
        f'{HEALTHZ_ENDPOINT}: Unable to validate response!',
        f'Got: {status}',
        'Expected: {"healthy": true}'
    )
    raise SystemExit(1)
