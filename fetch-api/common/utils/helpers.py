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


def time_since_now(past: str, past_tz: str = 'Europe/Sofia'):
    tz = zoneinfo.ZoneInfo(past_tz)
    past_dt = datetime.fromisoformat(past)

    if past_dt.tzinfo is None:
        past_dt = past_dt.replace(tzinfo=tz)

    now = datetime.now(tz)

    diff = now - past_dt
    seconds = diff.total_seconds()

    days = int(seconds // 86400)
    hours = int((seconds % 86400) // 3600)
    minutes = int((seconds % 3600) // 60)

    if days > 0:
        if days < 5:
            return f'{days}d{hours}h'
        else:
            return f'{days}d'

    elif hours > 0:
        if hours < 12:
            return f'{hours}h{minutes}m'
        else:
            return f'{hours}h'

    elif minutes > 0:
        return f'{minutes}m'

    else:
        return 'just now'
