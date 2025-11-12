import json
from common.utils.web import create_session
from common.messages.api import client_responses
from common.telemetry.src.tracing.wrappers import traced
from fetch_api.settings import (settings, connectors)
from fetch_api.src.telemetry.logging import log
from fetch_api.src.client import ConnectorClient
from fastapi.responses import JSONResponse



class APIProcessor:
    @staticmethod
    @traced()
    def process_request(
        request,
        client: ConnectorClient,
        upstream_method: str,
        upstream_endpoint: str,
        ai_prompt: str=None,
        ai_instructions_template: str='default',
        span=None
    ):
        common_log_attributes = {
            'connector': client.connector_name,
            'endpoint': request.scope['path'],
            'upstream_endpoint': upstream_endpoint
        }

        try:
            if upstream_method == 'GET':
                result = client.get(upstream_endpoint)

            elif upstream_method == 'POST':
                result = client.post(
                    endpoint=upstream_endpoint,
                    data=request.model_dump()
                )

            assert result.status_code in (200, 201)
            query_result = result.json()

            log.debug('Fetch completed', extra=common_log_attributes)

            if client.connector_name != 'ml' and request.ai:
                if 'ml' in connectors:
                    from fetch_api.src.routes.ml import client as ml_client
                    upstream_ml_endpoint = 'ask'

                    try:
                        response = ml_client.post(
                            endpoint=upstream_ml_endpoint,
                            data={
                                'instructions_template': ai_instructions_template,
                                'prompt': '{}\n\n{}'.format(
                                    ai_prompt,
                                    json.dumps(query_result['items'])
                                )
                            }
                        )

                        assert response.status_code == 200
                        query_result['ai_summary'] = response.json()['items'][0]

                        log.debug('Fetched AI summary', extra={
                            **common_log_attributes,
                            'upstream_ml_endpoint': upstream_ml_endpoint
                        })

                    except Exception as ai_err:
                        log.warning('AI summary fetch failed', extra={
                            **common_log_attributes,
                            'upstream_ml_endpoint': upstream_ml_endpoint,
                            'error': str(ai_err)
                        })

                else:
                    log.warning('Skipping AI processing, as ML connector is not enabled', extra={
                        **common_log_attributes,
                        'upstream_ml_endpoint': upstream_ml_endpoint
                    })

            return JSONResponse(content=query_result, status_code=200)

        except Exception as err:
            log.error('Fetch failed', extra={
                **common_log_attributes,
                'error': str(err)
            })

            return JSONResponse(content=client_responses['upstream-error'], status_code=502)
