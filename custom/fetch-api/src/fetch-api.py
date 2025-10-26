import requests, json
from os import getenv
from sys import argv
from flask import Flask, request, jsonify


class FetchAPI:
    def __init__(self, services=[]):
        self.services = services
        self.supported_services = {
            'grafana': {
                'required_vars': ['GRAFANA_URL', 'GRAFANA_SAT', 'GRAFANA_METRICS_DATASOURCE_UID']
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
                    'prometheus': getenv('GRAFANA_METRICS_DATASOURCE_UID')
                }
                self.grafana_headers = {
                    'Authorization': f'Bearer {self.grafana_token}',
                    'Accept': 'application/json',
                    'Content-Type': 'application/json'
                }

                self.grafana_queries_payload = self.read_file('lib/grafana/queries-payload.json')
                self.grafana_queries = self.read_file('lib/grafana/queries.json')

            self.verify_connection(service)


    def verify_connection(self, service):
        if service == 'grafana':
            connection = self.request(
                f'{self.grafana_url}/api/health',
                method='GET',
                headers=self.grafana_headers
            )

            if connection.status_code != 200:
                raise ConnectionError(f'{service}: [{connection.status_code}] Connection failed.')
            
            else:
                print(f'{service}: Connection successful.')


    def read_file(self, file, as_json=True):
        with open(file, 'r') as f:
            contents = f.read()

        if as_json:
            return json.loads(contents)

        return contents


    def request(self, url, method='GET', headers={}, data=None):
        response = None

        if method == 'GET':
            response = requests.get(url, headers=headers)

        elif method == 'POST':
            response = requests.post(url, headers=headers, data=data)

        elif method == 'PUT':
            response = requests.put(url, headers=headers, data=data)

        elif method == 'DELETE':
            response = requests.delete(url, headers=headers)

        return response


    def grafana(self, item):
        def argocd_apps(self):
            request_url = f'{self.grafana_url}/api/ds/query'
            request_payload = self.grafana_queries_payload['prometheus']['argocd-apps']
            request_payload['queries'][0]['datasource']['uid'] = self.grafana_datasources['prometheus']
            request_payload['queries'][0]['expr'] = self.grafana_queries['prometheus']['argocd-apps']['query']

            request_response = self.request(
                url = request_url,
                headers = self.grafana_headers,
                method = 'POST',
                data = json.dumps(request_payload)
            ).json()

            listener_response = []

            try:
                for item in request_response['results']['query']['frames']:
                    listener_response += [{
                        **item['schema']['fields'][1]['labels']
                    }]

                listener_response = sorted(
                    listener_response,
                    key = lambda x: (
                        x['health_status'] == 'Healthy',
                        x['name'].lower()
                    )
                )
            except:
                listener_response = [
                    'ERROR from fetch-api: grafana > argocd-apps is unable to parse response from Grafana API'
                ]

            return listener_response

        if item == 'argocd-apps':
            return argocd_apps(self)


class FetchAPIListener:
    def __init__(self):
        self.app = Flask('fetch-api')
        self.routes()


    def run(self, **kwargs):
        self.app.run(**kwargs)


    def routes(self):
        @self.app.route('/grafana/argocd-apps', methods=['GET'])
        def fetch_grafana_argocd_apps():
            return jsonify(
                fetch_api.grafana('argocd-apps')
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
