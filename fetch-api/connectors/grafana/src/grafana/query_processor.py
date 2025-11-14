from datetime import datetime
from common.utils.helpers import time_beautify_ms, time_since
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword



class Processor:
    @staticmethod
    @traced()
    def process(query_id: str, query_response: dict, span=None):
        result = []

        if query_id == 'argocd-apps':
            for item in query_response['results']['query']['frames']:
                labels = item['schema']['fields'][1]['labels']
                result += [{**labels}]

            result = sorted(
                result,
                key = lambda item: (
                    item['health_status'] == 'Healthy',
                    item['name'].lower()
                )
            )

        elif query_id == 'teslamate-usable-battery-level':
            result += [{
                'usable_battery_percentage': query_response['results']['query']['frames'][0]['data']['values'][0][0]
            }]

        elif query_id == 'teslamate-last-seen-location':
            last_changed = time_beautify_ms(query_response['results']['query']['frames'][0]['data']['values'][2][0])
            last_seen_since = time_since(
                past=last_changed,
                now=time_beautify_ms(datetime.now().strftime('%Y-%m-%dT%H:%M:%S'))
            )

            result += [{
                'city': query_response['results']['query']['frames'][0]['data']['values'][0][0],
                'address': query_response['results']['query']['frames'][0]['data']['values'][1][0],
                'last_changed': last_changed,
                'last_seen_since': last_seen_since
            }]

        elif query_id == 'teslamate-car-state':
            last_changed = time_beautify_ms(query_response['results']['query']['frames'][0]['data']['values'][1][0])
            state_since = time_since(
                past=last_changed,
                now=time_beautify_ms(datetime.now().strftime('%Y-%m-%dT%H:%M:%S'))
            )

            result += [{
                'state': query_response['results']['query']['frames'][0]['data']['values'][0][0],
                'last_changed': last_changed,
                'state_since': state_since
            }]

        elif query_id == 'teslamate-average-consumption':
            result += [{
                'average_consumption_wh_per_km': query_response['results']['query']['frames'][0]['data']['values'][0][0]
            }]

        span.set_attributes(
            reword({
                'processor.result.total_items': len(result),
                'processor.result.items': result
            })
        )

        return {
            'total_items': len(result),
            'items': result
        }
