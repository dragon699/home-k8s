from common.messages.api import client_responses
from fetch_api.settings import connectors
from fetch_api.src.telemetry.logging import log
from fetch_api.src.client import ConnectorClient
from fastapi import APIRouter
from fastapi.responses import JSONResponse

from fetch_api.src.schemas.ml import MLRequestAsk


router = APIRouter()
client = ConnectorClient(connectors['ml'].name)


@router.post('/ask')
def get_argocd_apps(request: MLRequestAsk):
    try:
        result = client.post('ask')

        assert result.status_code == 200

        log.debug('Fetch completed', extra={
            'connector': 'ml',
            'endpoint': '/ml/ask',
            'destination_endpoint': '/ask'
        })

        return JSONResponse(content=result.json(), status_code=200)

    except Exception as err:
        log.error('Fetch failed', extra={
            'connector': 'ml',
            'endpoint': '/ml/ask',
            'destination_endpoint': '/ask',
            'error': str(err)
        })

        return JSONResponse(content=client_responses['upstream-error'], status_code=502)
