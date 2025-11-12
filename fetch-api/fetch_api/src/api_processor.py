import json
from common.utils.web import create_session
from common.messages.api import client_responses
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword
from fetch_api.settings import (settings, connectors)
from fetch_api.src.telemetry.logging import log
from fetch_api.src.client import ConnectorClient
from fastapi.responses import JSONResponse



class APIProcessor:
    @staticmethod
    @traced()
    def process_request(client: ConnectorClient, request, endpoint: str, upstream_endpoint: str, ai_prompt: str, span=None):
        try:
            result = client.get(upstream_endpoint)
            
            assert result.status_code == 200
            query_result = result.json()

            log.debug('Fetch completed', extra={
                'connector': 'grafana',
                'endpoint': endpoint,
                'upstream_endpoint': upstream_endpoint
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
                                    ai_prompt,
                                    json.dumps(query_result['items'])
                                )
                            }
                        )

                        assert response.status_code == 200
                        query_result['ai_summary'] = response.json()['items'][0]

                        log.debug('Fetched AI summary', extra={
                            'connector': 'grafana',
                            'endpoint': endpoint,
                            'ml_endpoint': '/ml/ask',
                            'upstream_endpoint': upstream_endpoint
                        })

                    except Exception as ai_err:
                        log.warning('AI summary fetch failed', extra={
                            'connector': 'grafana',
                            'endpoint': endpoint,
                            'ml_endpoint': '/ml/ask',
                            'upstream_endpoint': upstream_endpoint,
                            'error': str(ai_err)
                        })

                else:
                    log.warning('Skipping AI processing, as ML connector is not enabled', extra={
                        'connector': 'grafana',
                        'endpoint': endpoint,
                        'ml_endpoint': '/ml/ask',
                        'upstream_endpoint': upstream_endpoint
                    })

            return JSONResponse(content=query_result, status_code=200)

        except Exception as err:
            log.error('Fetch failed', extra={
                'connector': 'grafana',
                'endpoint': endpoint,
                'upstream_endpoint': upstream_endpoint,
                'error': str(err)
            })

            return JSONResponse(content=client_responses['upstream-error'], status_code=502)
