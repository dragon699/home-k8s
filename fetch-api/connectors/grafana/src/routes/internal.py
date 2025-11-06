from fastapi import APIRouter
from connectors.grafana.settings import settings


router = APIRouter()


@router.get('/healthz')
def healthz():
    return {
        'connector_name': settings.name,
        'healthy': settings.healthy,
        'health_endpoint': settings.health_endpoint,
        'health_last_check': settings.health_last_check,
        'health_next_check': settings.health_next_check
    }
