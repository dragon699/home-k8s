from fastapi import FastAPI
from apscheduler.schedulers.background import BackgroundScheduler
from contextlib import asynccontextmanager

from fetch_api.settings import (settings, connectors)
from fetch_api.src.telemetry.tracing import instrumentor
from fetch_api.src.health_checker import HealthChecker
from fetch_api.src.loaders import RoutesLoader


@asynccontextmanager
async def lifespan(app: FastAPI):
    scheduler.start()

    for connector_name in connectors:
        health_checkers[connector_name] = HealthChecker(
            scheduler,
            connectors[connector_name]
        )
        health_checkers[connector_name].create_connector_schedule()

    RoutesLoader.load(app, settings)

    yield

    scheduler.shutdown()


scheduler = BackgroundScheduler()
health_checkers = {}

app = FastAPI(lifespan=lifespan)
instrumentor.instrument(app)
