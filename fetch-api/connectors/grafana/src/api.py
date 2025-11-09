from fastapi import FastAPI
from apscheduler.schedulers.background import BackgroundScheduler
from contextlib import asynccontextmanager

from connectors.grafana.settings import settings
from connectors.grafana.src.telemetry.tracing import instrumentor
from connectors.grafana.src.grafana.client import GrafanaClient
from connectors.grafana.src.health_checker import HealthChecker
from connectors.grafana.src.loaders import RoutesLoader


@asynccontextmanager
async def lifespan(app: FastAPI):
    scheduler.start()
    health_checker.create_schedule()
    grafana_client.authenticate()

    RoutesLoader.load(app, settings)

    yield

    scheduler.shutdown()


scheduler = BackgroundScheduler()
health_checker = HealthChecker(scheduler)
grafana_client = GrafanaClient()

app = FastAPI(lifespan=lifespan)
instrumentor.instrument(app)
