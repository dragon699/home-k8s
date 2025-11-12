import json, yaml
import os
import zoneinfo
from datetime import datetime, timezone


def read_file(path: str, type='json'):
    if not os.path.exists(path):
        return None

    with open(path, 'r') as file:
        if type == 'json':
            return json.load(file)

        elif type in ('yaml', 'yml'):
            return yaml.safe_load(file)

        else:
            return file.read()


def beautify_ms(milliseconds: int, target_tz: str = 'Europe/Sofia'):
    seconds = milliseconds / 1000
    tz = zoneinfo.ZoneInfo(target_tz)
    
    dt_utc = datetime.fromtimestamp(seconds, tz=timezone.utc)
    dt_sofia = dt_utc.astimezone(tz)

    return dt_sofia.strftime('%Y-%m-%dT%H:%M:%S')
