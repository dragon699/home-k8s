from common.messages.api import client_responses
from connectors.ml.src.ollama.querier import querier
from connectors.ml.src.telemetry.logging import log
from fastapi import APIRouter
from fastapi.responses import JSONResponse

from connectors.ml.src.schemas.ollama import RequestAsk


router = APIRouter()


@router.post('/ask', tags=['ollama'], summary='Ask Ollama models a question')
def ask_ollama(request: RequestAsk) -> JSONResponse:
    try:
        result = querier.commit(
            prompt=request.prompt,
            model=request.model,
            instructions=request.instructions,
            instructions_template=request.instructions_template
        )

        assert not result is None

        log.info('Query executed successfully')

        return JSONResponse(content=result, status_code=200)

    except Exception as err:
        log.error('Query execution failed', extra={
            'error': str(err)
        })

        return JSONResponse(content=client_responses['server-error'], status_code=500)
