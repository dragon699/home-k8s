from fastapi import FastAPI
from apscheduler.schedulers.background import BackgroundScheduler
from contextlib import asynccontextmanager

from connectors.ml.settings import settings
from connectors.ml.src.telemetry.tracing import instrumentor
from connectors.ml.src.ollama.client import OllamaClient
from connectors.ml.src.health_checker import HealthChecker
from connectors.ml.src.loaders import RoutesLoader


@asynccontextmanager
async def lifespan(app: FastAPI):
    scheduler.start()
    health_checker.create_schedule()

    RoutesLoader.load(app, settings)

    yield

    scheduler.shutdown()


scheduler = BackgroundScheduler()
health_checker = HealthChecker(scheduler)
ollama_client = OllamaClient()

app = FastAPI(lifespan=lifespan)
instrumentor.instrument(app)
