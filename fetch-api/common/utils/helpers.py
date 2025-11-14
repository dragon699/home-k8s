import zoneinfo
from datetime import datetime, timezone
from common.utils.system import read_file



def get_app_version(version_file: str):
    try:
        return read_file(version_file, type=None).strip()

    except:
        print(f'{version_file} not found, traces will not have version info.')
        return 'unknown'


def time_beautify_ms(milliseconds: int, target_tz: str = 'Europe/Sofia'):
    seconds = milliseconds / 1000
    tz = zoneinfo.ZoneInfo(target_tz)
    
    dt_utc = datetime.fromtimestamp(seconds, tz=timezone.utc)
    dt_sofia = dt_utc.astimezone(tz)

    return dt_sofia.strftime('%Y-%m-%dT%H:%M:%S')


def time_since(past: str, now: str):
    past = datetime.fromisoformat(past)
    now = datetime.fromisoformat(now)

    diff = now - past
    seconds = diff.total_seconds()

    hours = int(seconds // 3600)
    minutes = int((seconds % 3600) // 60)

    if hours > 0:
        if hours < 3:
            return f'{hours}h{minutes}m'
        else:
            return f'{hours}h'
    
    elif minutes > 0:
        return f'{minutes}m'
    
    else:
        return 'just now'
