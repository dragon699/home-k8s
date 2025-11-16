from fetch_api.settings import connectors
from fetch_api.src.client import ConnectorClient
from fetch_api.src.api_processor import APIProcessor
from fastapi import APIRouter, Request

from fetch_api.src.schemas.ml import MLBody


router = APIRouter()
client = ConnectorClient(
    connectors['ml'].name,
    requests_timeout=20,
    cache=True
)


@router.post('/ask')
def fetch_ai_summary(
    request: Request,
    body: MLBody
):
    return APIProcessor.process_request(
        request=request,
        body=body,
        client=client,
        upstreams=[{
            'method': 'POST',
            'endpoint':'ask'
        }]
    )
