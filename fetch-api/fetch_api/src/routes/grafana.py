from fetch_api.settings import connectors
from fetch_api.src.client import ConnectorClient
from fetch_api.src.api_processor import APIProcessor
from fastapi import APIRouter

from fetch_api.src.schemas.grafana import GrafanaRequest


router = APIRouter()
client = ConnectorClient(connectors['grafana'].name)


@router.post('/argocd-apps')
def fetch_argocd_apps(request: GrafanaRequest):
    return APIProcessor.process_request(
        client=client,
        request=request,
        endpoint='/grafana/argocd-apps',
        upstream_endpoint='prometheus/argocd-apps',
        ai_prompt='How are my ArgoCD apps doing?'
    )


@router.post('/car-battery')
def fetch_car_battery(request: GrafanaRequest):
    return APIProcessor.process_request(
        client=client,
        request=request,
        endpoint='/grafana/car-battery',
        upstream_endpoint='postgresql/car-battery',
        ai_prompt='How much battery does my Tesla have, do I have to charge soon?'
    )


@router.post('/car-last-location')
def fetch_car_last_location(request: GrafanaRequest):
    return APIProcessor.process_request(
        client=client,
        request=request,
        endpoint='/grafana/car-last-location',
        upstream_endpoint='postgresql/car-last-location',
        ai_prompt='Where was my car seen last?'
    )
