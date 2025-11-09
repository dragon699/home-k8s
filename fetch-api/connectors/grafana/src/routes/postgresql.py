from common.messages.api import client_responses
from connectors.grafana.src.grafana.querier import querier
from connectors.grafana.src.telemetry.logging import log
from fastapi import APIRouter
from fastapi.responses import JSONResponse


router = APIRouter()


@router.get('/car-battery')
def argocd_apps():
    try:
        result = querier.commit(
            query_ds_type='postgresql',
            query_id='teslamate-usable-battery-level'
        )

        assert not result is None

        log.info('Query executed successfully', extra={
            'query_ds_type': 'postgresql',
            'query_id': 'teslamate-usable-battery-level'
        })

        return JSONResponse(content=result, status_code=200)

    except Exception as err:
        log.error('Query execution failed', extra={
            'query_ds_type': 'postgresql',
            'query_id': 'teslamate-usable-battery-level',
            'error': str(err)
        })

        return JSONResponse(content=client_responses['server-error'], status_code=500)
