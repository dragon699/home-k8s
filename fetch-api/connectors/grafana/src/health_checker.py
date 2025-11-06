from datetime import datetime, timedelta
from connectors.grafana.src.telemetry.logging import log
from connectors.grafana.settings import settings

from opentelemetry import trace
from common.utils.web import create_session
from common.telemetry.src.tracing.span_wrapper import traced



class HealthChecker:
    def __init__(self, scheduler):
        self.scheduler = scheduler

    
    @traced()
    def ping_grafana(self, span=None):
        session = create_session(timeout=5)
        response = session.get(settings.health_endpoint)
        return response
    

    @traced()
    def get_grafana_health(self, span=None):
        def get_next_interval():
            if settings.healthy is True:
                return settings.health_check_interval_seconds

            else:
                return settings.health_retry_interval_seconds

        was_healthy = settings.healthy
        settings.health_last_check = datetime.now().isoformat().split('.')[0]

        get_next_check_time = lambda: (
            datetime.now() + timedelta(seconds=get_next_interval())
        ).isoformat().split('.')[0]

        get_common_attr = lambda: {
            'health_endpoint': settings.health_endpoint,
            'last_check': settings.health_last_check,
            'next_check': settings.health_next_check
        }

        try:
            response = self.ping_grafana()

            if response.status_code == 200:
                settings.healthy = True

                if was_healthy is not True:
                    self.schedule_grafana_check()

                else:
                    settings.health_next_check = get_next_check_time()

                if was_healthy is not True:
                    log.debug('Health check successful', extra={
                        **get_common_attr(),
                        'healthy': True,
                        'status_code': response.status_code
                    })

            else:
                span.set_attributes({
                    'error.message': response.text,
                    'error.status_code': response.status_code
                })

                settings.healthy = False

                if was_healthy is not False:
                    self.schedule_grafana_check()
                
                else:
                    settings.health_next_check = get_next_check_time()

                log.warning('Health check failed', extra={
                    **get_common_attr(),
                    'healthy': False,
                    'status_code': response.status_code
                })

        except Exception as err:
            span.set_attributes({
                "error.message": str(err)
            })

            settings.healthy = None

            if was_healthy is not None:
                self.schedule_grafana_check()

            else:
                settings.health_next_check = get_next_check_time()

            log.error(f'Health check failed, Grafana unreachable', extra={
                **get_common_attr(),
                'healthy': None,
                'error': str(err)
            })


    @traced()
    def schedule_grafana_check(self, span=None):
        if not settings.health_job_id is None:
            try:
                self.scheduler.remove_job(settings.health_job_id)

            except:
                pass

        if settings.healthy is True:
            interval_seconds = settings.health_check_interval_seconds

        else:
            interval_seconds = settings.health_retry_interval_seconds

        settings.health_next_check = (
            datetime.now() + timedelta(seconds=interval_seconds)
        ).isoformat().split('.')[0]

        job = self.scheduler.add_job(
            self.get_grafana_health,
            'interval',
            seconds=interval_seconds,
            id=f'health_check_grafana'
        )

        settings.health_job_id = job.id

        span.set_attributes({
            'next_check': settings.health_next_check,
            'last_check': settings.health_last_check,
            'interval_seconds': interval_seconds
        })


def check_health(scheduler):
    checker = HealthChecker(scheduler)

    checker.schedule_grafana_check()
    checker.get_grafana_health()
