from common.messages.api import client_responses
from fetch_api.settings import connectors
from fetch_api.src.telemetry.logging import log
from fetch_api.src.client import ConnectorClient
from fastapi import APIRouter
from fastapi.responses import JSONResponse


router = APIRouter()
client = ConnectorClient(connectors['grafana'].name)


@router.get('/argocd-apps')
def get_argocd_apps():
    try:
        result = client.get('prometheus/argocd-apps')

        assert result.status_code == 200

        log.debug('Fetch completed', extra={
            'connector': 'grafana',
            'endpoint': '/grafana/argocd-apps',
            'destination_endpoint': '/prometheus/argocd-apps'
        })

        return JSONResponse(content=result.json(), status_code=200)

    except Exception as err:
        log.error('Fetch failed', extra={
            'connector': 'grafana',
            'endpoint': '/grafana/argocd-apps',
            'destination_endpoint': '/prometheus/argocd-apps',
            'error': str(err)
        })

        return JSONResponse(content=client_responses['upstream-error'], status_code=502)


@router.get('/car-battery')
def get_car_battery():
    try:
        result = client.get('postgresql/car-battery')

        assert result.status_code == 200

        log.debug('Fetch completed', extra={
            'connector': 'grafana',
            'endpoint': '/car-battery',
            'destination_endpoint': '/postgresql/car-battery'
        })

        return JSONResponse(content=result.json(), status_code=200)

    except Exception as err:
        log.error('Fetch failed', extra={
            'connector': 'grafana',
            'endpoint': '/argocd-apps',
            'destination_endpoint': '/prometheus/argocd-apps',
            'error': str(err)
        })

        return JSONResponse(content=client_responses['upstream-error'], status_code=502)
