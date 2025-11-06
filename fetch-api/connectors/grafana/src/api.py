from fastapi import FastAPI
from apscheduler.schedulers.background import BackgroundScheduler
from contextlib import asynccontextmanager

from connectors.grafana.src.telemetry.logging import log
from connectors.grafana.src.telemetry.tracing import instrumentor
from connectors.grafana.src.health_checker import check_health
from connectors.grafana.src.loaders import RoutesLoader


scheduler = BackgroundScheduler()

@asynccontextmanager
async def lifespan(app: FastAPI):
    log.debug(f'Scheduling Grafana health checks..')
    scheduler.start()

    check_health(scheduler)

    yield

    log.debug('Terminating Grafana health check schedule..')
    scheduler.shutdown()


app = FastAPI(lifespan=lifespan)
instrumentor.instrument(app)

RoutesLoader.load(app)
