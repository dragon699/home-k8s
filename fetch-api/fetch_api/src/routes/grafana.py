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
        upstream_method='GET',
        upstream_endpoint='prometheus/argocd-apps',
        ai_prompt='How are my ArgoCD apps doing?',
        ai_instructions_template='default'
    )


@router.post('/car-battery')
def fetch_car_battery(request: Request, body: GrafanaBody):
    return APIProcessor.process_request(
        request=request,
        body=body,
        client=client,
        upstream_method='GET',
        upstream_endpoint='postgresql/car-battery',
        ai_prompt='How much battery does my Tesla have, do I have to charge soon?',
        ai_instructions_template='default'
    )


@router.post('/car-last-location')
def fetch_car_last_location(request: Request, body: GrafanaBody):
    return APIProcessor.process_request(
        request=request,
        body=body,
        client=client,
        upstream_method='GET',
        upstream_endpoint='postgresql/car-last-location',
        ai_prompt='Where was my car seen last?',
        ai_instructions_template='default'
    )
