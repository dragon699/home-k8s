from datetime import datetime
from common.utils.helpers import time_beautify_ms, time_since_now
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword



class Processor:
    @staticmethod
    @traced()
    def map(query_response: dict, span=None):
        result = {}

        frame = query_response['results']['query']['frames'][0]
        fields = frame['schema']['fields']
        values = frame['data']['values']

        for index, field in enumerate(fields):
            result[field['name']] = values[index][0]

        return result
    

    @staticmethod
    @traced()
    def rename(query_id: str, data: dict, span=None):
        result = {}
        reword_map = {
            'teslamate-usable-battery-level': {
                'usable_battery_level': 'usable_battery_percentage',
                'kwh': 'usable_battery_kwh'
            },
            'teslamate-last-charge-info': {
                'date': 'last_charge',
                'type': 'charge_type',
                'energy_added': 'charge_energy_added_kwh',
                'start_percent': 'charge_start_percentage',
                'end_percent': 'charge_end_percentage'
            },
            'teslamate-last-seen-location': {
                'time': 'last_seen'
            },
            'teslamate-car-state': {
                'since': 'last_updated'
            },
            'teslamate-average-consumption': {
                'wh_per_km': 'average_consumption_wh_per_km'
            }
        }

        if not query_id in reword_map:
            return data

        for key in data:
            if key in reword_map[query_id]:
                result[reword_map[query_id][key]] = data[key]
            else:
                result[key] = data[key]

        return result


    @staticmethod
    @traced()
    def process(query_id: str, query_response: dict, span=None):
        result = []

        if query_id.startswith('teslamate-'):
            print(query_response)
            data = Processor.map(query_response)
            data = Processor.rename(query_id, data)

            if len(data) > 0:
                if query_id == 'teslamate-usable-battery-level':
                    data['usable_battery_kwh'] = round(data['usable_battery_kwh'], 2)
                    result += [data]

                elif query_id == 'teslamate-last-charge-info':
                    data['last_charge'] = time_beautify_ms(data['last_charge'])
                    data['last_charge_since'] = time_since_now(data['last_charge'])
                    data['charge_energy_added_percentage'] = data['charge_end_percentage'] - data['charge_start_percentage']
                    result += [data]

                elif query_id == 'teslamate-last-seen-location':
                    data['last_seen'] = time_beautify_ms(data['last_seen'])
                    data['last_seen_since'] = time_since_now(data['last_seen'])
                    result += [data]
                    

                elif query_id == 'teslamate-car-state':
                    data['last_updated'] = time_beautify_ms(data['last_updated'])
                    data['last_state_since'] = time_since_now(data['last_updated'])
                    result += [data]

                elif query_id == 'teslamate-average-consumption':
                    result += [data]


        else:
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
