from fetch_api.settings import connectors
from fetch_api.src.client import ConnectorClient
from fetch_api.src.api_processor import APIProcessor
from fastapi import APIRouter, Request, Query

from fetch_api.src.schemas.grafana import GrafanaBody


router = APIRouter()
client = ConnectorClient(connectors['grafana'].name)


@router.post('/argocd-apps')
def fetch_argocd_apps(
    request: Request,
    body: GrafanaBody
):
    return APIProcessor.process_request(
        request=request,
        body=body,
        client=client,
        upstreams=[{
            'method': 'GET',
            'endpoint':'prometheus/argocd-apps'
        }],
        ai_prompt='Should I be worried about the state of my ArgoCD applications?',
        ai_instructions_template='default'
    )


@router.post('/car-info')
def fetch_car_info(
    request: Request,
    body: GrafanaBody
):
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
                'postgresql/car-efficiency'
            ]
        ],
        ai_prompt='Give me a summary for my Tesla.',
        ai_instructions_template='default'
    )


@router.post('/car-drives-history')
def fetch_car_drives_history(
    request: Request,
    body: GrafanaBody,
    number_of_drives: int = Query(5, ge=1, le=1000)
):
    return APIProcessor.process_request(
        request=request,
        body=body,
        client=client,
        upstreams=[{
            'method': 'GET',
            'endpoint':'postgresql/car-drives-history',
            'params': {
                'number_of_drives': number_of_drives
            }
        }],
        ai_prompt=f'Provide an analysis of my last {number_of_drives} drives history with my Tesla.',
        ai_instructions_template='default'
    )
