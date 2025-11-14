from fetch_api.settings import connectors
from fetch_api.src.client import ConnectorClient
from fetch_api.src.api_processor import APIProcessor
from fastapi import APIRouter, Request

from fetch_api.src.schemas.grafana import GrafanaBody


router = APIRouter()
client = ConnectorClient(connectors['grafana'].name)


@router.post('/argocd-apps')
def fetch_argocd_apps(request: Request, body: GrafanaBody):
    return APIProcessor.process_request(
        request=request,
        body=body,
        client=client,
        upstreams=[{
            'method': 'GET',
            'endpoint':'prometheus/argocd-apps'
        }],
        ai_prompt='How are my ArgoCD apps doing?',
        ai_instructions_template='default'
    )


@router.post('/car-info')
def fetch_car_info(request: Request, body: GrafanaBody):
    return APIProcessor.process_request(
        request=request,
        body=body,
        client=client,
        upstreams=[
            {
                'method': 'GET',
                'endpoint': endpoint
            } for endpoint in [
                'postgresql/car-battery',
                'postgresql/car-last-charge',
                'postgresql/car-last-location',
                'postgresql/car-state',
                'postgresql/car-average-consumption'
            ]
        ],
        ai_prompt='Give me a summary for my Tesla.',
        ai_instructions_template='default'
    )
