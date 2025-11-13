from common.utils.system import beautify_ms
from common.telemetry.src.tracing.wrappers import traced



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
            result += [{
                'city': query_response['results']['query']['frames'][0]['data']['values'][0][0],
                'address': query_response['results']['query']['frames'][0]['data']['values'][1][0],
                'time': beautify_ms(query_response['results']['query']['frames'][0]['data']['values'][2][0])
            }]

        elif query_id == 'teslamate-car-state':
            result += [{
                'state': query_response['results']['query']['frames'][0]['data']['values'][0][0]
            }]

        span.set_attributes({
            'processor.result.total_items': len(result),
            'processor.result.items': result
        })

        return {
            'total_items': len(result),
            'items': result
        }
