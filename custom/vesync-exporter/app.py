import asyncio
import json
import logging
import os
import signal
import time

import aiohttp
from prometheus_client import Counter, Gauge, start_http_server
from prometheus_client.core import REGISTRY, GaugeMetricFamily
from pyvesync import VeSync
from pyvesync.const import ConnectionStatus, DeviceStatus, ProductTypes
from pyvesync.utils.errors import ErrorCodes, VeSyncLoginError, VeSyncRateLimitError


EMAIL = os.environ['VESYNC_EMAIL']
PASSWORD = os.environ['VESYNC_PASSWORD']
COUNTRY_CODE = os.getenv('VESYNC_COUNTRY_CODE', 'US')
POLL_INTERVAL = int(os.getenv('POLL_INTERVAL_SECONDS', '60'))
DEVICE_REFRESH_INTERVAL = int(os.getenv('DEVICE_REFRESH_INTERVAL_SECONDS', '3600'))
BACKOFF_INTERVAL = int(os.getenv('BACKOFF_INTERVAL_SECONDS', '300'))
REQUEST_TIMEOUT = int(os.getenv('REQUEST_TIMEOUT_SECONDS', '30'))
PORT = int(os.getenv('PORT', '8000'))

logging.basicConfig(
    level=os.getenv('LOG_LEVEL', 'INFO').upper(),
    format='%(asctime)s %(levelname)s %(name)s: %(message)s'
)
log = logging.getLogger('vesync-exporter')

STATE_GAUGES = {
    'pm25': ('vesync_purifier_pm25_ugm3', 'PM2.5 concentration in ug/m3'),
    'pm1': ('vesync_purifier_pm1_ugm3', 'PM1.0 concentration in ug/m3'),
    'pm10': ('vesync_purifier_pm10_ugm3', 'PM10 concentration in ug/m3'),
    'air_quality_level': ('vesync_purifier_air_quality_level', 'Air quality level, 1 excellent to 4 poor'),
    'aq_percent': ('vesync_purifier_air_quality_percent', 'Air quality percentage'),
    'voc': ('vesync_purifier_voc', 'VOC reading'),
    'co2': ('vesync_purifier_co2_ppm', 'CO2 concentration in ppm'),
    'humidity': ('vesync_purifier_humidity_percent', 'Relative humidity'),
    'temperature': ('vesync_purifier_temperature', 'Temperature as reported by the device'),
    'filter_life': ('vesync_purifier_filter_life_percent', 'Remaining filter life'),
    'fan_level': ('vesync_purifier_fan_level', 'Fan level as reported by the cloud, only the live speed in manual mode'),
    'fan_set_level': ('vesync_purifier_fan_set_level', 'Configured manual fan level'),
    'auto_room_size': ('vesync_purifier_auto_room_size_sqft', 'Room size used by auto mode in sq ft'),
    'nightlight_brightness': ('vesync_purifier_nightlight_brightness_percent', 'Night light brightness'),
    'fan_rotate_angle': ('vesync_purifier_fan_rotate_angle_degrees', 'Fan oscillation angle'),
}

STATE_BOOLS = {
    'device_status': ('vesync_purifier_power_on', 'Whether the purifier is switched on'),
    'display_status': ('vesync_purifier_display_on', 'Whether the display is currently lit'),
    'display_set_status': ('vesync_purifier_display_configured_on', 'Whether the display is enabled in settings'),
    'display_forever': ('vesync_purifier_display_forever', 'Whether the display stays on permanently'),
    'child_lock': ('vesync_purifier_child_lock', 'Whether child lock is enabled'),
    'light_detection_switch': ('vesync_purifier_light_detection_enabled', 'Whether light detection is enabled'),
    'light_detection_status': ('vesync_purifier_light_detected', 'Whether light is currently detected'),
}

RAW_GAUGES = {
    'device_error_code': ('vesync_purifier_error_code', 'Device error code reported by the cloud, Core 400S units report 11802000 while working normally'),
    'extension.timer_remain': ('vesync_purifier_timer_remaining_seconds', 'Seconds left on the auto-off timer'),
    'extension.schedule_count': ('vesync_purifier_schedules', 'Number of configured schedules'),
}

STATE_RAW_SOURCES = {
    'fan_set_level': 'levelNew',
}

RAW_CONSUMED = {
    'enabled', 'filter_life', 'mode', 'level', 'levelNew', 'air_quality', 'air_quality_value',
    'display', 'child_lock', 'night_light', 'configuration.display', 'configuration.display_forever',
    'configuration.auto_preference.room_size', 'configuration.auto_preference.type', *RAW_GAUGES,
}

LOGINS = Counter('vesync_logins_total', 'Logins performed against the VeSync cloud')
ERRORS = Counter('vesync_errors_total', 'Failed VeSync API interactions', ['type'])
UP = Gauge('vesync_up', 'Whether the last poll cycle completed without errors')
LAST_SUCCESS = Gauge('vesync_last_success_timestamp_seconds', 'Time of the last fully successful poll cycle')
POLL_DURATION = Gauge('vesync_poll_duration_seconds', 'Duration of the last poll cycle')


class Collector:
    def __init__(self):
        self.samples = []

    def collect(self):
        families = {}
        for name, doc, labels, value in self.samples:
            family = families.get(name)
            if family is None:
                family = families[name] = GaugeMetricFamily(name, doc, labels=list(labels))
            family.add_metric(list(labels.values()), value)
        yield from families.values()


def as_bool(value):
    if isinstance(value, bool):
        return value
    if isinstance(value, str) and value.lower() in (DeviceStatus.ON, DeviceStatus.OFF):
        return value.lower() == DeviceStatus.ON
    return None


def flatten(data, prefix=''):
    for key, value in data.items():
        path = f'{prefix}{key}'
        if isinstance(value, dict):
            yield from flatten(value, f'{path}.')
        else:
            yield path, value


def inner_result(device):
    response = getattr(device.last_response, 'response_data', None) or {}
    result = response.get('result')
    if isinstance(result, dict) and isinstance(result.get('result'), dict):
        return result['result']
    return result if isinstance(result, dict) else {}


def one_hot(add, name, doc, label, options, current):
    for option in dict.fromkeys([*map(str, options), str(current)]):
        if option and option != 'unknown':
            add(name, doc, option == str(current), **{label: option})


def info_samples(device):
    labels = {
        'device': device.device_name,
        'cid': device.cid,
        'device_type': device.device_type,
        'mac': device.mac_id or '',
        'region': device.device_region or '',
        'connection_type': device.connection_type or '',
        'firmware': device.current_firm_version or '',
        'latest_firmware': device.latest_firm_version or '',
    }
    samples = [('vesync_device_info', 'Static VeSync device information', labels, 1.0)]
    if device.latest_firm_version:
        samples.append((
            'vesync_device_firmware_update_available', 'Whether a newer firmware is available',
            {'device': device.device_name}, float(device.firmware_update)
        ))
    return samples


def state_samples(device, raw):
    base = {'device': device.device_name}
    samples = []

    def add(name, doc, value, /, **extra):
        samples.append((name, doc, {**base, **extra}, float(value)))

    state = device.state
    fields = dict(flatten(raw))
    for attr, (name, doc) in STATE_GAUGES.items():
        if attr in STATE_RAW_SOURCES and STATE_RAW_SOURCES[attr] not in fields:
            continue
        value = getattr(state, attr, None)
        if isinstance(value, (int, float)) and not isinstance(value, bool) and value >= 0:
            add(name, doc, value)
    for attr, (name, doc) in STATE_BOOLS.items():
        value = as_bool(getattr(state, attr, None))
        if value is not None:
            add(name, doc, value)

    one_hot(add, 'vesync_purifier_mode', 'Active purifier mode', 'mode', device.modes, state.mode)
    if state.auto_preference_type:
        one_hot(
            add, 'vesync_purifier_auto_preference', 'Active auto mode preference', 'preference',
            device.auto_preferences, state.auto_preference_type
        )
    if state.nightlight_status:
        one_hot(
            add, 'vesync_purifier_nightlight_mode', 'Active night light mode', 'mode',
            device.nightlight_modes, state.nightlight_status
        )

    for path, (name, doc) in RAW_GAUGES.items():
        value = fields.get(path)
        if isinstance(value, (int, float)):
            add(name, doc, value)
    error_code = fields.get('device_error_code')
    if isinstance(error_code, int) and error_code != 0:
        error = ErrorCodes.get_error_info(error_code)
        add(
            'vesync_purifier_error_info', 'Decoded device error reported by the cloud', 1,
            code=str(error_code), name=error.name, message=error.message
        )
    for path, value in fields.items():
        if path not in RAW_CONSUMED and isinstance(value, (int, float)):
            add('vesync_purifier_status_field', 'Other numeric fields of the raw purifier status', value, field=path)

    add('vesync_device_last_update_timestamp_seconds', 'Time the device status was last fetched', time.time())
    return samples


class Exporter:
    def __init__(self, manager, collector):
        self.manager = manager
        self.collector = collector
        self.next_device_refresh = 0.0
        self.logged_raw = set()

    async def ensure_login(self):
        if self.manager.enabled:
            return
        LOGINS.inc()
        if not await self.manager.login():
            raise VeSyncLoginError('Login returned False')
        log.info('Logged in to VeSync (%s)', self.manager.current_region)

    async def refresh_devices(self):
        if time.monotonic() < self.next_device_refresh:
            return
        await self.manager.get_devices()
        self.next_device_refresh = time.monotonic() + DEVICE_REFRESH_INTERVAL
        purifiers = self.purifiers()
        log.info('Found %d purifier(s): %s', len(purifiers), ', '.join(
            f'{d.device_name} ({d.device_type})' for d in purifiers
        ))
        try:
            await self.manager.check_firmware()
        except Exception as e:
            ERRORS.labels(type=type(e).__name__).inc()
            log.warning('Firmware check failed: %s', e)

    def purifiers(self):
        return [d for d in self.manager.devices if d.product_type == ProductTypes.PURIFIER]

    async def poll_device(self, device):
        samples = info_samples(device)
        ok = True
        try:
            await device.update()
        except (VeSyncLoginError, VeSyncRateLimitError):
            raise
        except Exception as e:
            ERRORS.labels(type=type(e).__name__).inc()
            log.warning('Updating %s failed: %s', device.device_name, e)
            ok = False

        online = device.state.connection_status == ConnectionStatus.ONLINE
        fetched = ok and online and getattr(device.last_response, 'code', None) in (0, None)
        if ok and not online:
            log.info('%s is offline', device.device_name)
        elif ok and not fetched:
            ERRORS.labels(type='DeviceResponseError').inc()
            log.warning('%s returned %s', device.device_name, device.last_response.message)
            ok = False

        samples.append(('vesync_device_online', 'Whether the device is connected to the VeSync cloud', {'device': device.device_name}, float(online)))
        samples.append(('vesync_device_up', 'Whether the last status fetch for the device succeeded', {'device': device.device_name}, float(fetched)))

        if fetched:
            raw = inner_result(device)
            if device.cid not in self.logged_raw:
                log.info('Raw status of %s: %s', device.device_name, json.dumps(raw, default=str))
                self.logged_raw.add(device.cid)
            samples.extend(state_samples(device, raw))
        return ok, samples

    async def poll(self):
        await self.ensure_login()
        await self.refresh_devices()
        results = await asyncio.gather(*(self.poll_device(d) for d in self.purifiers()))
        self.collector.samples = [s for _, samples in results for s in samples]
        return all(ok for ok, _ in results)

    async def run(self, stop):
        while not stop.is_set():
            started = time.monotonic()
            delay = POLL_INTERVAL
            try:
                ok = await self.poll()
            except VeSyncRateLimitError:
                ERRORS.labels(type='VeSyncRateLimitError').inc()
                log.warning('Rate limited by VeSync, backing off for %ss', BACKOFF_INTERVAL)
                ok, delay = False, BACKOFF_INTERVAL
            except VeSyncLoginError as e:
                ERRORS.labels(type='VeSyncLoginError').inc()
                log.error('Login failed: %s, retrying in %ss', e, BACKOFF_INTERVAL)
                self.manager.enabled = False
                ok, delay = False, BACKOFF_INTERVAL
            except Exception as e:
                ERRORS.labels(type=type(e).__name__).inc()
                log.exception('Poll failed')
                ok = False

            UP.set(ok)
            if ok:
                LAST_SUCCESS.set_to_current_time()
            POLL_DURATION.set(time.monotonic() - started)

            try:
                await asyncio.wait_for(stop.wait(), timeout=max(0.0, delay - (time.monotonic() - started)))
            except TimeoutError:
                pass


async def main():
    collector = Collector()
    REGISTRY.register(collector)
    start_http_server(PORT)
    log.info('Serving metrics on :%d, polling every %ss', PORT, POLL_INTERVAL)

    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, stop.set)

    timeout = aiohttp.ClientTimeout(total=REQUEST_TIMEOUT)
    async with aiohttp.ClientSession(timeout=timeout) as session:
        async with VeSync(EMAIL, PASSWORD, country_code=COUNTRY_CODE, session=session) as manager:
            await Exporter(manager, collector).run(stop)


if __name__ == '__main__':
    asyncio.run(main())
