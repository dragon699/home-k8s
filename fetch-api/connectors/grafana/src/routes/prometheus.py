from common.messages.api import client_responses
from connectors.grafana.src.grafana.querier import querier
from connectors.grafana.src.telemetry.logging import log
from fastapi import APIRouter
from fastapi.responses import JSONResponse


router = APIRouter()


@router.get('/argocd-apps')
def argocd_apps():
    try:
        result = querier.commit(
            query_ds_type='prometheus',
            query_id='argocd-apps'
        )

        assert not result is None

        log.info('Query executed successfully', extra={
            'query_ds_type': 'prometheus',
            'query_id': 'argocd-apps'
        })

        return JSONResponse(content=result, status_code=200)

    except Exception as err:
        log.error('Query execution failed', extra={
            'query_ds_type': 'prometheus',
            'query_id': 'argocd-apps',
            'error': str(err)
        })

        return JSONResponse(content=client_responses['server-error'], status_code=500)
