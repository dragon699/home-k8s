from datetime import (datetime, timedelta)
from fetch_api.src.telemetry.logging import log
from fetch_api.settings import (settings, connectors)

from common.utils.web import create_session
from common.telemetry.src.tracing.wrappers import traced



class HealthChecker:
    def __init__(self, scheduler, connector):
        self.scheduler = scheduler
        self.connector = connector


    def get_next_interval(self):
        if self.connector.healthy is True:
            return settings.connector_health_check_interval_seconds

        else:
            return settings.connector_health_retry_interval_seconds


    def get_next_check_time(self):
        return (
            datetime.now() + timedelta(seconds=self.get_next_interval())
        ).isoformat().split('.')[0]


    @traced()
    def ping_connector(self, span=None):
        session = create_session(timeout=5)
        response = session.get(self.connector.health_endpoint)
        return response


    @traced()
    def get_connector_status(self, span=None):
        was_healthy = self.connector.healthy
        self.connector.health_last_check = datetime.now().isoformat().split('.')[0]

        get_common_attr = lambda: {
            'connector': self.connector.name,
            'health_endpoint': self.connector.health_endpoint,
            'last_check': self.connector.health_last_check,
            'next_check': self.connector.health_next_check
        }

        try:
            response = self.ping_connector()

        except Exception as err:
            span.set_attributes({
                'error.connector': self.connector.name,
                'error.message': str(err)
            })

            self.connector.healthy = None

            if was_healthy is not None:
                self.update_connector_schedule()

            else:
                self.connector.health_next_check = self.get_next_check_time()
            
            log.critical(f'Health check failed, connector unreachable', extra={
                **get_common_attr(),
                'healthy': None,
                'error': str(err)
            })

            return None
        
        try:
            is_healthy = response.json()['healthy']

        except Exception as err:
            span.set_attributes({
                'error.connector': self.connector.name,
                'error.message': str(err)
            })

            self.connector.healthy = None

            if was_healthy is not None:
                self.update_connector_schedule()

            else:
                self.connector.health_next_check = self.get_next_check_time()

            log.critical('Health check failed, connector reachable, but not functioning correctly', extra={
                **get_common_attr(),
                'healthy': None,
                'status_code': response.status_code,
                'error': str(err)
            })

            return None

        if isinstance(is_healthy, bool):
            if (response.status_code == 200) and (is_healthy is True):
                self.connector.healthy = True

                if was_healthy is not True:
                    self.update_connector_schedule()

                else:
                    self.connector.health_next_check = self.get_next_check_time()
                
                if was_healthy is not True:
                    log.debug('Health check successful', extra={
                        **get_common_attr(),
                        'healthy': True,
                        'status_code': response.status_code
                    })

            else:
                span.set_attributes({
                    'error.connector': self.connector.name,
                    'error.message': response.text,
                    'error.status_code': response.status_code
                })

                self.connector.healthy = False

                if was_healthy is not False:
                    self.update_connector_schedule()

                else:
                    self.connector.health_next_check = self.get_next_check_time()
                
                log.critical('Health check failed', extra={
                    **get_common_attr(),
                    'healthy': False,
                    'status_code': response.status_code
                })

        elif is_healthy is None:
            span.set_attributes({
                'error.connector': self.connector.name,
                'error.message': response.text
            })

            self.connector.healthy = None

            if was_healthy is not None:
                self.update_connector_schedule()

            else:
                self.connector.health_next_check = self.get_next_check_time()

            log.critical('Health check failed, connector reachable, but not functioning correctly', extra={
                **get_common_attr(),
                'healthy': None,
                'status_code': response.status_code
            })


    @traced()
    def create_connector_schedules(scheduler, span=None):
        for conn_name in connectors:
            checker = HealthChecker(scheduler, connectors[conn_name])

            checker.update_connector_schedule()
            checker.get_connector_status()


    @traced()
    def update_connector_schedule(self, span=None):
        if not self.connector.health_job_id is None:
            try:
                self.scheduler.remove_job(self.connector.health_job_id)

            except:
                pass

        interval_seconds = self.get_next_interval()

        self.connector.health_next_check = (
            datetime.now() + timedelta(seconds=interval_seconds)
        ).isoformat().split('.')[0]

        job = self.scheduler.add_job(
            self.get_connector_status,
            'interval',
            seconds=interval_seconds,
            id=f'health_check_{self.connector.name}'
        )

        self.connector.health_job_id = job.id
        span.set_attributes({
            'next_check': self.connector.health_next_check,
            'interval_seconds': interval_seconds,
            **(
                {'last_check': self.connector.health_last_check} if self.connector.health_last_check else {}
            )
        })
