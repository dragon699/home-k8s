from fastapi import FastAPI
from apscheduler.schedulers.background import BackgroundScheduler
from contextlib import asynccontextmanager

from fetch_api.src.telemetry.logging import log
from fetch_api.src.telemetry.tracing import instrumentor
from fetch_api.src.health_checker import HealthChecker
from fetch_api.src.loaders import RoutesLoader


@asynccontextmanager
async def lifespan(app: FastAPI):
    log.debug(f'Scheduling connector health checks..')
    scheduler.start()

    HealthChecker.create_connector_schedules(scheduler)
    RoutesLoader.load(app)

    yield

    log.debug('Terminating connector health check schedules..')
    scheduler.shutdown()


scheduler = BackgroundScheduler()

app = FastAPI(lifespan=lifespan)
instrumentor.instrument(app)
