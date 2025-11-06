import json, urllib.request
from os import getenv


URL = f'http://localhost:{getenv("LISTEN_PORT")}'
ENDPOINT = 'api/healthz'

HEALTHZ_ENDPOINT = f'{URL}/{ENDPOINT}'


try:
    status = json.loads(
        urllib.request.urlopen(
            HEALTHZ_ENDPOINT
        ).read()
    )

except:
    print(f'{HEALTHZ_ENDPOINT}: Unreachable!')
    raise SystemExit(1)


try:
    if status['healthy'] is True:
        raise SystemExit(0)
    
    else:
        print(f'{HEALTHZ_ENDPOINT}: Unhealthy!')
        raise SystemExit(1)

except (KeyError, TypeError):
    print(
        f'{HEALTHZ_ENDPOINT}: Unable to validate response!',
        f'Got: {status}',
        'Expected: {"healthy": true}'
    )
    raise SystemExit(1)
