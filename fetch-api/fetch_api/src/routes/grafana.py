import json
from common.utils.web import create_session
from common.messages.api import client_responses
from fetch_api.settings import (settings, connectors)
from fetch_api.src.telemetry.logging import log
from fetch_api.src.client import ConnectorClient
from fastapi import APIRouter
from fastapi.responses import JSONResponse

from fetch_api.src.schemas.grafana import GrafanaRequest


router = APIRouter()
client = ConnectorClient(connectors['grafana'].name)


@router.post('/argocd-apps')
def fetch_argocd_apps(request: GrafanaRequest):
    try:
        result = client.get('prometheus/argocd-apps')

        assert result.status_code == 200
        query_result = result.json()

        log.debug('Fetch completed', extra={
            'connector': 'grafana',
            'endpoint': '/grafana/argocd-apps',
            'destination_endpoint': '/prometheus/argocd-apps'
        })

        if request.ai:
            if 'ml' in connectors:
                try:
                    session = create_session(timeout=180)
                    response = session.post(
                        f'http://{settings.listen_host}:{settings.listen_port}/ml/ask',
                        headers={'Content-Type': 'application/json'},
                        json={
                            'instructions_template': 'default',
                            'prompt': '{}\n\n{}'.format(
                                'How are my argocd apps doing?',
                                json.dumps(query_result['items'])
                            )
                        }
                    )

                    assert response.status_code == 200
                    query_result['ai_summary'] = response.json()['items'][0]

                    log.debug('Fetched AI summary', extra={
                        'connector': 'grafana',
                        'endpoint': '/grafana/argocd-apps',
                        'ai_endpoint': '/ml/ask',
                        'destination_endpoint': '/prometheus/argocd-apps'
                    })

                except Exception as ai_err:
                    log.warning('AI summary fetch failed', extra={
                        'connector': 'grafana',
                        'endpoint': '/grafana/argocd-apps',
                        'ai_endpoint': '/ml/ask',
                        'destination_endpoint': '/prometheus/argocd-apps',
                        'error': str(ai_err)
                    })

            else:
                log.warning('Skipping AI processing, as ML connector is not enabled', extra={
                    'connector': 'grafana',
                    'endpoint': '/grafana/argocd-apps',
                    'ai_endpoint': '/ml/ask',
                    'destination_endpoint': '/prometheus/argocd-apps'
                })

        return JSONResponse(content=query_result, status_code=200)

    except Exception as err:
        log.error('Fetch failed', extra={
            'connector': 'grafana',
            'endpoint': '/grafana/argocd-apps',
            'destination_endpoint': '/prometheus/argocd-apps',
            'error': str(err)
        })

        return JSONResponse(content=client_responses['upstream-error'], status_code=502)


@router.post('/car-battery')
def fetch_car_battery(request: GrafanaRequest):
    try:
        result = client.get('postgresql/car-battery')

        assert result.status_code == 200
        query_result = result.json()

        log.debug('Fetch completed', extra={
            'connector': 'grafana',
            'endpoint': '/grafana/car-battery',
            'destination_endpoint': '/postgresql/car-battery'
        })

        if request.ai:
            if 'ml' in connectors:
                try:
                    session = create_session(timeout=180)
                    response = session.post(
                        f'http://{settings.listen_host}:{settings.listen_port}/ml/ask',
                        headers={'Content-Type': 'application/json'},
                        json={
                            'instructions_template': 'default',
                            'prompt': '{}\n\n{}'.format(
                                'How much battery does my Tesla have, do I need to charge soon?',
                                json.dumps(query_result['items'])
                            )
                        }
                    )

                    assert response.status_code == 200
                    query_result['ai_summary'] = response.json()['items'][0]

                    log.debug('Fetched AI summary', extra={
                        'connector': 'grafana',
                        'endpoint': '/grafana/car-battery',
                        'ai_endpoint': '/ml/ask',
                        'destination_endpoint': '/postgresql/car-battery'
                    })

                except Exception as ai_err:
                    log.warning('AI summary fetch failed', extra={
                        'connector': 'grafana',
                        'endpoint': '/grafana/car-battery',
                        'ai_endpoint': '/ml/ask',
                        'destination_endpoint': '/postgresql/car-battery',
                        'error': str(ai_err)
                    })

            else:
                log.warning('Skipping AI processing, as ML connector is not enabled', extra={
                    'connector': 'grafana',
                    'endpoint': '/grafana/car-battery',
                    'ai_endpoint': '/ml/ask',
                    'destination_endpoint': '/postgresql/car-battery'
                })

        return JSONResponse(content=query_result, status_code=200)

    except Exception as err:
        log.error('Fetch failed', extra={
            'connector': 'grafana',
            'endpoint': '/grafana/car-battery',
            'destination_endpoint': '/postgresql/car-battery',
            'error': str(err)
        })

        return JSONResponse(content=client_responses['upstream-error'], status_code=502)
