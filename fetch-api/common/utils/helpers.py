import zoneinfo
from datetime import datetime, timezone
from common.utils.system import read_file



def get_app_version(version_file: str):
    try:
        return read_file(version_file, type=None).strip()

    except:
        print(f'{version_file} not found, traces will not have version info.')
        return 'unknown'
    

def get_maps_url(path: str):
    if not path.startswith('new?lat='):
        return path
    
    try:
        lat = path.split('lat=')[1].split('&')[0]
        lng = path.split('lng=')[1]

        return f'https://google.com/maps/search/?api=1&query={lat},{lng}'
    
    except:
        return path
    

def get_maps_directions_url(start_url: str, end_url: str):
    extract = lambda q: q.split('query=')[1]

    if not start_url.startswith('new?lat=') or not end_url.startswith('new?lat='):
        return 'N/A'
    
    try:
        start = extract(start_url)
        end = extract(end_url)

        return f'https://www.google.com/maps/dir/?api=1&origin={start}&destination={end}'
    
    except:
        return 'N/A'
    

def get_teslamate_drive_grafana_url(drive_id: int, drive_start_time: str, drive_end_time: str):
    grafana_url = 'https://grafana.k8s.iaminyourpc.xyz'
    grafana_dashboard_path = 'd/zm7wN6Zgz/driving-details'

    return f'{grafana_url}/{grafana_dashboard_path}?from={drive_start_time}&to={drive_end_time}&var-drive_id={drive_id}&timezone=Europe%2FSofia&orgId=1&var-temp_unit=C&var-length_unit=km&var-alternative_length_unit=m&var-preferred_range=rated&var-base_url=https:%2F%2Fcar.k8s.iaminyourpc.xyz&var-pressure_unit=bar&var-speed_unit=km%2Fh'


def time_beautify_ms(milliseconds: int, target_tz: str = 'Europe/Sofia', convert_tz: bool = True):
    seconds = milliseconds / 1000
    tz = zoneinfo.ZoneInfo(target_tz)
    
    dt_utc = datetime.fromtimestamp(seconds, tz=timezone.utc)

    if convert_tz:
        dt = dt_utc.astimezone(tz)
    else:
        dt = dt_utc

    return dt.strftime('%Y-%m-%dT%H:%M:%S')


def time_beautify_ordinal(dt_string: int, target_tz: str = 'Europe/Sofia'):
    def get_ordinal_day(n: int):
        if 10 <= n % 100 <= 20:
            suffix = 'th'
        else:
            suffix = {1: 'st', 2: 'nd', 3: 'rd'}.get(n % 10, 'th')
        return f'{n}{suffix}'
    
    tz = zoneinfo.ZoneInfo(target_tz)
    dt = datetime.fromisoformat(dt_string)

    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=tz)

    day = get_ordinal_day(dt.day)
    month = dt.strftime('%B')
    year = dt.year
    time = dt.strftime('%H:%M')

    return f'{year} / {day} of {month} at {time}'


def time_now(target_tz: str = 'Europe/Sofia'):
    tz = zoneinfo.ZoneInfo(target_tz)
    now = datetime.now(tz)

    return now.strftime('%Y-%m-%dT%H:%M:%S')


def time_since(past: str, future: str = None, tz: str = 'Europe/Sofia', instant: bool = True):
    tz = zoneinfo.ZoneInfo(tz)
    past_dt = datetime.fromisoformat(past)

    if past_dt.tzinfo is None:
        past_dt = past_dt.replace(tzinfo=tz)
    
    if future is None:
        future_dt = datetime.now(tz)
    
    else:
        future_dt = datetime.fromisoformat(future)

        if future_dt.tzinfo is None:
            future_dt = future_dt.replace(tzinfo=tz)

    diff = future_dt - past_dt
    seconds = diff.total_seconds()

    days = int(seconds // 86400)
    hours = int((seconds % 86400) // 3600)
    minutes = int((seconds % 3600) // 60)

    if days > 0:
        if days < 7:
            return f'{days}d{hours}h'
        else:
            if instant:
                return f'{days}d'
            else:
                return f'{days}d+'

    elif hours > 0:
        if hours < 24:
            return f'{hours}h{minutes}m'
        else:
            if instant:
                return f'{hours}h'
            else:
                return f'{hours}h+'

    elif minutes > 0:
        return f'{minutes}m'

    else:
        if instant:
            return 'just now'
        else:
            return '<1m'
