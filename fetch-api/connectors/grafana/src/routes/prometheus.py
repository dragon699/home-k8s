from connectors.grafana.src.api_processor import APIProcessor
from fastapi import APIRouter


router = APIRouter()


@router.get('/argocd-apps', tags=['prometheus'], summary='Retrieve ArgoCD applications and their statuses')
def get_argocd_apps() -> dict:
    return APIProcessor.process_request(
        query_ds_type='prometheus',
        query_id='argocd-apps'
    )
