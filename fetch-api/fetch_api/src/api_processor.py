import json
from fastapi.responses import JSONResponse
from common.telemetry.src.tracing.wrappers import traced
from fetch_api.settings import connectors, settings
from fetch_api.src.telemetry.logging import log
from fetch_api.src.client import ConnectorClient



class APIProcessor:
    @staticmethod
    @traced()
    def process_request(
        request,
        body,
        client: ConnectorClient,
        upstreams: list[dict[str, str]],
        ai_prompt: str = None,
        ai_instructions_template: str = 'default',
        span=None
    ):
        status_code = None
        results = {'total_items': 0, 'items': []}
        common_log_attributes = {
            'connector': client.connector_name,
            'endpoint': request.scope['path']
        }

        for upstream in upstreams:
            common_stream_log_attributes = {
                **common_log_attributes.copy(),
                'upstream_endpoint': upstream['endpoint']
            }

            try:
                upstream_method = upstream['method']
                upstream_endpoint = upstream['endpoint']
                params = upstream.get('params', {})

                if upstream_method == 'GET':
                    response = client.get(
                        endpoint=upstream_endpoint,
                        params=params
                    )

                elif upstream_method == 'POST':
                    response = client.post(
                        endpoint=upstream_endpoint,
                        params=params,
                        data=body.model_dump(
                            exclude={'ai'}
                        )
                    )

                assert response.status_code in (200, 201)
                
                response_body = response.json()
                results['items'] += response_body['items']
                upstreams[upstreams.index(upstream)]['status'] = 'success'

                log.debug('Upstream fetch completed', extra=common_stream_log_attributes)

            except Exception as err:
                upstreams[upstreams.index(upstream)]['status'] = 'failed'
                log.warning('Upstream fetch failed', extra={
                    **common_stream_log_attributes,
                    'error': str(err)
                })

        if client.connector_name != 'ml':
            if body.ai and len(results['items']) > 0:
                if 'ml' in connectors:
                    from fetch_api.src.client import ConnectorClient

                    ml_client = ConnectorClient(
                        connectors['ml'].name,
                        cache=True,
                        requests_timeout=settings.ai_summary_requests_timeout
                    )

                    upstream_ml_endpoint = 'ask'
                    commong_ml_log_attributes = {
                        **common_log_attributes,
                        'upstream_ml_endpoint': upstream_ml_endpoint
                    }

                    try:
                        response = ml_client.post(
                            endpoint=upstream_ml_endpoint,
                            data={
                                'instructions_template': ai_instructions_template,
                                'prompt': '{}\n\n{}'.format(
                                    ai_prompt,
                                    json.dumps(results['items'])
                                )
                            }
                        )

                        assert response.status_code == 200

                        results['ai_summary'] = response.json()['items'][0]

                        if response.json().get('cached') and response.json()['cached'] is True:
                            commong_ml_log_attributes['cache_status'] = 'hit'
                            results['ai_summary']['cached'] = True
                            results['ai_summary']['cached_at'] = response.json()['cached_at']

                        else:
                            commong_ml_log_attributes['cache_status'] = 'miss'

                        log.debug('Fetched AI summary for upstreams response', extra=commong_ml_log_attributes)

                    except Exception as err:
                        log.warning('AI summary fetch failed', extra={
                            **commong_ml_log_attributes,
                            'error': str(err)
                        })

                else:
                    log.warning('Skipping AI processing, as ML connector is not enabled', extra=common_log_attributes)

        results['total_items'] = len(results['items'])

        if any(
            upstream['status'] == 'failed'
            for upstream in upstreams
        ):
            status_code = 502

        else:
            status_code = 200

        log.info('Fetch completed', extra=common_log_attributes)

        return JSONResponse(
            status_code=status_code,
            content=results
        )
