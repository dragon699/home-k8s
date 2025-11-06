from fastapi import FastAPI
from apscheduler.schedulers.background import BackgroundScheduler
from contextlib import asynccontextmanager

from fetch_api.src.telemetry.logging import log
from fetch_api.src.telemetry.tracing import instrumentor
from fetch_api.src.health_checker import check_health
from fetch_api.src.loaders import RoutesLoader


scheduler = BackgroundScheduler()

@asynccontextmanager
async def lifespan(app: FastAPI):
    log.debug(f'Scheduling connector health checks..')
    scheduler.start()

    check_health(scheduler)

    yield

    log.debug('Terminating connector health check schedules..')
    scheduler.shutdown()


app = FastAPI(lifespan=lifespan)
instrumentor.instrument(app)

RoutesLoader.load(app)
