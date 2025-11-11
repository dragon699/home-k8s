from connectors.ml.settings import settings
from fastapi import APIRouter


router = APIRouter()


@router.get('/health')
def health():
    return {
        'connector_name': settings.name,
        'healthy': settings.healthy,
        'health_endpoint': settings.health_endpoint,
        'health_last_check': settings.health_last_check,
        'health_next_check': settings.health_next_check
    }


@router.get('/ready')
def ready():
    return {
        'ready': settings.healthy
    }
