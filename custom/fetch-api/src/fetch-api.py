from lib.instrumentation import *

instrumentation = FetchAPIInstrumentation(**otel_meta)
tracer = instrumentation.get_tracer()

from lib.common import *
from os import getenv
from sys import argv
from flask import Flask, request, jsonify
import json



class FetchAPI:
    def __init__(self, services=[]):
        self.services = services
        self.supported_services = {
            'grafana': {
                'required_vars': [
                    'GRAFANA_URL',
                    'GRAFANA_SAT',
                    'GRAFANA_METRICS_DATASOURCE_UID',
                    'GRAFANA_TESLAMATE_DATASOURCE_UID'
                ]
            }
        }
        self.disabled_services = []

        self.verify_config()


    def verify_config(self):
        if len(self.services) == 0:
            raise ValueError(f'At least one service must be specified. Supported services are {list(self.supported_services.keys())}')

        print('Provided services:')
        print(self.services)

        for service in self.services:
            if not service in self.supported_services:
                raise ValueError(f'{service}: Unsupported service, supported services are {list(self.supported_services.keys())}')

            print(f'{service}: Establishing connection..')

            required_vars = []

            if 'required_vars' in self.supported_services[service]:
                required_vars = self.supported_services[service]['required_vars']

                if len(required_vars) > 0:
                    for var in required_vars:
                        if getenv(var) is None:
                            raise ValueError(f'{service}: Missing required environment variable: {var}')

            if service == 'grafana':
                self.grafana_url = getenv('GRAFANA_URL')
                self.grafana_token = getenv('GRAFANA_SAT')
                self.grafana_datasources = {
                    'prometheus': getenv('GRAFANA_METRICS_DATASOURCE_UID'),
                    'teslamate': getenv('GRAFANA_TESLAMATE_DATASOURCE_UID')
                }
                self.grafana_headers = {
                    'Authorization': f'Bearer {self.grafana_token}',
                    'Accept': 'application/json',
                    'Content-Type': 'application/json'
                }

                self.grafana_queries_payload = read_file('lib/grafana/queries-payload.json')
                self.grafana_queries = read_file('lib/grafana/queries.json')

            self.verify_connection(service)


    def verify_connection(self, service):
        if service == 'grafana':
            connection = make_request(
                f'{self.grafana_url}/api/health',
                method='GET',
                headers=self.grafana_headers
            )

            if connection.status_code != 200:
                raise ConnectionError(f'{service}: [{connection.status_code}] Connection failed.')
            
            else:
                print(f'{service}: Connection successful.')


    def grafana(self, item):
        def argocd_apps(self):
            s_grafana.add_event('build service request')

            request_url = f'{self.grafana_url}/api/ds/query'
            request_payload = self.grafana_queries_payload['prometheus']['argocd-apps']
            request_payload['queries'][0]['datasource']['uid'] = self.grafana_datasources['prometheus']
            request_payload['queries'][0]['expr'] = self.grafana_queries['prometheus']['argocd-apps']['query']

            s_grafana.add_event('send service request', {
                'http.url': request_url,
                'http.method': 'POST',
                'http.payload': json.dumps(request_payload, indent=2)[:512],
                'fetch-api.grafana.argocd_apps.query': request_payload['queries'][0]['expr']
            })

            request_response = make_request(
                url = request_url,
                headers = self.grafana_headers,
                method = 'POST',
                data = json.dumps(request_payload)
            )

            if request_response.status_code != 200:
                s_grafana.add_event('error receiving service response', {
                    'http.status_code': request_response.status_code,
                    'http.response': json.dumps(request_response.text, indent=2)[:512]
                })
                s_grafana.set_status(StatusCode.ERROR)

                return [f'ERROR from fetch-api: FetchAPI.grafana.argocd_apps Grafana request failed!']

            s_grafana.add_event('receive service response', {
                'http.status_code': request_response.status_code,
                'http.response': json.dumps(request_response.text, indent=2)[:512]
            })

            listener_response = []

            s_grafana.add_event('extract data')

            try:
                for item in request_response.json()['results']['query']['frames']:
                    labels = item['schema']['fields'][1]['labels']

                    listener_response += [{
                        label: labels[label]
                        for label in labels
                        if label in self.grafana_queries['prometheus']['argocd-apps']['exported_fields']
                    }]

                listener_response = sorted(
                    listener_response,
                    key = lambda x: (
                        x['health_status'] == 'Healthy',
                        x['name'].lower()
                    )
                )

                s_grafana.add_event('finalize', {
                    'fetch-api.grafana.argocd_apps.count': len(listener_response),
                    'fetch-api.grafana.argocd_apps.list': json.dumps(listener_response, indent=2)[:512]
                })

            except:
                listener_response = ['ERROR from fetch-api: FetchAPI.grafana.argocd_apps is unable to parse response from Grafana API!']
                s_grafana.add_event('error extracting data', {
                    'fetch-api.grafana.argocd_apps.error': json.dumps(listener_response, indent=2)[:512]
                })
                s_grafana.set_status(StatusCode.ERROR)

            return listener_response


        def car_battery(self):
            s_grafana.add_event('build service request')

            request_url = f'{self.grafana_url}/api/ds/query'
            request_payload = self.grafana_queries_payload['postgresql']['car-battery']
            request_payload['queries'][0]['datasource']['uid'] = self.grafana_datasources['teslamate']
            request_payload['queries'][0]['rawSql'] = self.grafana_queries['postgresql']['car-battery']['query']

            s_grafana.add_event('send service request', {
                'http.url': request_url,
                'http.method': 'POST',
                'http.payload': json.dumps(request_payload, indent=2)[:512],
                'fetch-api.grafana.car_battery.query': request_payload['queries'][0]['rawSql']
            })

            request_response = make_request(
                url=request_url,
                headers=self.grafana_headers,
                method='POST',
                data=json.dumps(request_payload)
            )

            if request_response.status_code != 200:
                s_grafana.add_event('error receiving service response', {
                    'http.status_code': request_response.status_code,
                    'http.response': json.dumps(request_response.text, indent=2)[:512]
                })
                s_grafana.set_status(StatusCode.ERROR)

                return ['ERROR from fetch-api: FetchAPI.grafana.car_battery Grafana request failed!']

            s_grafana.add_event('receive service response', {
                'http.status_code': request_response.status_code,
                'http.response': json.dumps(request_response.text, indent=2)[:512]
            })

            listener_response = []

            s_grafana.add_event('extract data')

            try:
                listener_response += [{
                    'usable_battery_percentage': request_response.json()['results']['query']['frames'][0]['data']['values'][0][0]
                }]

                s_grafana.add_event('finalize', {
                    'fetch-api.grafana.car_battery.usable_battery_percentage': listener_response[0]['usable_battery_percentage']
                })

            except:
                listener_response = ['ERROR from fetch-api: FetchAPI.grafana.car_battery is unable to parse response from Grafana API!']
                s_grafana.add_event('error extracting data', {
                    'fetch-api.grafana.car_battery.error': json.dumps(listener_response, indent=2)[:512]
                })
                s_grafana.set_status(StatusCode.ERROR)

            return listener_response


        executor = {
            'argocd-apps': lambda self: argocd_apps(self),
            'car-battery': lambda self: car_battery(self)
        }

        with tracer.start_as_current_span(f'FetchAPI.grafana.{item.replace("-", "_")}') as s_grafana:
            s_grafana.set_attributes({
                'fetch-api.service.name': 'grafana',
                'fetch-api.service.destination': item,
                'fetch-api.service.url': self.grafana_url
            })
            return executor[item](self)



class FetchAPIListener:
    def __init__(self):
        self.app = Flask('fetch-api')

        instrumentation.instrument_requests(self.app)
        self.routes()


    def run(self, **kwargs):
        self.app.run(**kwargs)


    def routes(self):
        @self.app.route('/grafana/argocd-apps', methods=['GET'])
        def fetch_grafana_argocd_apps():
            return jsonify(
                fetch_api.grafana('argocd-apps')
            )

        @self.app.route('/grafana/car-battery', methods=['GET'])
        def fetch_grafana_car_battery():
            return jsonify(
                fetch_api.grafana('car-battery')
            )


def verify_args():
    if len(argv) < 2:
        raise ValueError(f'Usage: python3 {argv[0]} <service1> <service2> ..')

    return argv[1:]


if __name__ == '__main__':
    services = verify_args()

    fetch_api = FetchAPI(services=services)
    fetch_api_listener = FetchAPIListener()

    fetch_api_listener.run(
        host = '0.0.0.0',
        port = 8080,
        debug = False
    )
