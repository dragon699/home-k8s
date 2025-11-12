from common.messages.api import client_responses
from fetch_api.settings import connectors
from fetch_api.src.telemetry.logging import log
from fetch_api.src.client import ConnectorClient
from fetch_api.src.api_processor import APIProcessor
from fastapi import APIRouter

from fetch_api.src.schemas.ml import MLRequest


router = APIRouter()
client = ConnectorClient(connectors['ml'].name)


@router.post('/ask')
def fetch_ai_summary(request: MLRequest):
    return APIProcessor.process_request(
        request=request,
        client=client,
        upstream_method='POST',
        upstream_endpoint='ask'
    )
