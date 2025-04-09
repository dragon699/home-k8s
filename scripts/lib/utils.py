# Common functions utilized by other scripts;
# such as for rendering config templates, creating cluster config;
# for k0sctl and importing Grafana dashboards;

import argparse, jinja2, yaml, json, base64
import requests, paramiko
import warnings
from os import chmod, makedirs, listdir, path, walk
from os import environ as env



class Common:
    @staticmethod
    def read_yaml(file_path):
        with open(file_path, 'r') as file:
            return yaml.full_load(file)

    @staticmethod
    def render_template(file_path, vars={}):
        with open(file_path, 'r') as file:
            template = jinja2.Template(
                file.read(),
                undefined=jinja2.StrictUndefined
            )

        return template.render(vars)

    @staticmethod
    def run_ssh_script_remotely(host, port, user, key, script_content, script_name='unset'):
        ssh_client = paramiko.SSHClient()
        ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

        try:
            print(f'[{script_name} @ {host}] Running script..')

            ssh_client.connect(
                hostname=host, port=port, username=user, key_filename=key
            )

            stdin, stdout, stderr = ssh_client.exec_command(script_content)

            stdout = stdout.read().decode('utf-8')
            stderr = stderr.read().decode('utf-8')

            print(f'[{script_name} @ {host}] Output:')
            print(stdout, '\n', stderr)

        finally:
            ssh_client.close()

    @staticmethod
    def api_get(url, headers={}):
        response = requests.get(
            url, headers=headers, verify=False
        )

        return [
            response.status_code, json.loads(response.text)
        ]

    @staticmethod
    def api_post(url, headers={}, data={}):
        headers = {
            'Content-Type': 'application/json',
            **headers
        }

        response = requests.post(
            url, headers=headers, data=json.dumps(data), verify=False
        )

        return [
            response.status_code, json.loads(response.text)
        ]
    
    @staticmethod
    def api_patch(url, headers={}, data={}):
        headers = {
            'Content-Type': 'application/json',
            **headers
        }

        response = requests.patch(
            url, headers=headers, data=json.dumps(data), verify=False
        )

        return [
            response.status_code, json.loads(response.text)
        ]


class CreateCluster:
    @staticmethod
    def create_k0sctl_config():
        try:
            values = Common.read_yaml(env['K0SCTL_CONFIG_INPUTS_PATH'])
            config = Common.render_template(env['K0SCTL_CONFIG_TEMPLATE_PATH'], values)

            makedirs(env['K0SCTL_CLUSTER_DIR'], exist_ok=True)

            with open(env["K0SCTL_CONFIG_PATH"], 'w') as file:
                file.write(config)

        except:
            raise Exception('Failed to render k0sctl config!')

    @staticmethod
    def create_destroy_script():
        values = {
            'K0SCTL_CONFIG_PATH': env['K0SCTL_CONFIG_PATH'],
            'K0SCTL_CLUSTER_DIR': env['K0SCTL_CLUSTER_DIR'],
            'HOME': env['HOME'],
        }

        content = Common.render_template(env['K0SCTL_DESTROY_SCRIPT_TEMPLATE_PATH'], values)

        with open(env['K0SCTL_DESTROY_SCRIPT_PATH'], 'w') as file:
            file.write(content)

        chmod(env['K0SCTL_DESTROY_SCRIPT_PATH'], 0o755)

    @staticmethod
    def get_flux_params():
        params_string = ''
        mappings = {
            'repository_ssh_url': '--url',
            'repository_ssh_private_key_path': '--private-key-file',
            'repository_branch': '--branch',
            'repository_dir_path': '--path',
        }

        values = Common.read_yaml(env['K0SCTL_CONFIG_INPUTS_PATH'])

        for value in mappings:
            if (value in values) and (not values[value] in [None, '']):
                params_string += f'{mappings[value]} {values[value]} '

            else:
                raise Exception(f'{env["K0SCTL_CONFIG_INPUTS_PATH"]}: {value} is missing, or it has an invalid value!')

        print(params_string)

    @staticmethod
    def apply_vm_kubernetes_prerequisites():
        try:
            values = Common.read_yaml(env['K0SCTL_CONFIG_INPUTS_PATH'])

            hosts = [
                {
                    'host': host['address'],
                    'port': host['port'],
                    'user': host['user'],
                    'key': host['keyPath'],
                } for host in values['cluster_nodes']
            ]

        except:
            raise Exception(f'{env["K0SCTL_CONFIG_INPUTS_PATH"]}: hosts not properly defined!')

        with open(env['REMOTE_SCRIPT'], 'r') as file:
            content = file.read()

        for host in hosts:
            Common.run_ssh_script_remotely(
                **host,
                script_content=content,
                script_name=env['REMOTE_SCRIPT'].split('/')[-1]
            )


class Grafana:
    @staticmethod
    def import_dashboards():
        def replace_datasource_uid(data, data_sources):
            if isinstance(data, dict):
                for key, value in data.items():
                    if key == 'datasource' and 'type' in value:
                        ds_type = value['type']

                        if ds_type in data_sources:
                            data[key]['uid'] = data_sources[ds_type]

                    else:
                        replace_datasource_uid(value, data_sources)

            elif isinstance(data, list):
                for item in data:
                    replace_datasource_uid(item, data_sources)

        warnings.filterwarnings("ignore", category=requests.packages.urllib3.exceptions.InsecureRequestWarning)

        for env_var in [
            'GRAFANA_URL', 'GRAFANA_DASHBOARDS_DIR',
            'GRAFANA_USER', 'GRAFANA_PASSWORD',
        ]:
            if not env_var in env or env[env_var] in [None, '']:
                raise Exception(f'{env_var}: missing or invalid environment variable!')

        api_url = f'{env["GRAFANA_URL"]}/api'

        api_credentials = base64.b64encode(
            f'{env["GRAFANA_USER"]}:{env["GRAFANA_PASSWORD"]}'.encode('utf-8')
        ).decode('utf-8')

        api_headers = {
            'Authorization': f'Basic {api_credentials}',
        }

        auth_check = Common.api_get(f'{api_url}/users', api_headers)

        if auth_check[0] == 401:
            raise Exception(f'{env["GRAFANA_URL"]}: Authentication failed!')

        data_sources = {
            ds['type']: ds['uid']
            for ds in Common.api_get(f'{api_url}/datasources', api_headers)[1]
        }

        try:
            for root, dirs, files in walk(env['GRAFANA_DASHBOARDS_DIR']):
                relative_path = path.relpath(root, env['GRAFANA_DASHBOARDS_DIR'])

                for file in files:
                    file_path = path.join(root, file)

                    if not file.endswith('.json'):
                        print(f' [{file}] not a JSON file, so skipping..')
                        continue

                    with open(file_path, 'r') as file:
                        dashboard_content = json.load(file)

                    replace_datasource_uid(dashboard_content, data_sources)

                    if relative_path == '.':
                        folder_id = None

                    else:
                        dir_name = path.basename(root)

                        existing_folders = Common.api_get(f'{api_url}/folders', api_headers)[1]
                        existing_folder_names = [folder['title'] for folder in existing_folders]

                        if dir_name in existing_folder_names:
                            print(f' [{dir_name}] using existing folder in Grafana')

                            folder_id = [
                                folder['id']
                                for folder in existing_folders
                                if folder['title'] == dir_name
                            ][0]

                        else:
                            folder_post_data = {
                                'title': dir_name,
                                'uid': '_'.join(dir_name.split()).lower(),
                            }

                            subfolders = relative_path.split('/')

                            if len(subfolders) > 1:
                                folder_post_data['parentUid'] = '_'.join(subfolders[-2].split()).lower()

                            created_folder = Common.api_post(
                                f'{api_url}/folders', api_headers, folder_post_data
                            )

                            if created_folder[0] == 200:
                                print(f' [{dir_name}] created folder in Grafana')
                                folder_id = created_folder[1]['id']

                    dashboard_post_data = {
                        'dashboard': {
                            **dashboard_content,
                            'id': None,
                        },
                        'overwrite': True,
                    }

                    if not folder_id in [None, '']:
                        dashboard_post_data['folderId'] = folder_id
                    
                    imported_dashboard = Common.api_post(
                        f'{api_url}/dashboards/import', api_headers, dashboard_post_data
                    )

                    if imported_dashboard[0] == 200:
                        print(f' [{dashboard_content["title"]}] imported dashboard to Grafana')
                    
                    else:
                        print(f' [{dashboard_content["title"]}] failed to import dashboard to Grafana')

                    if dashboard_content['title'] == env['GRAFANA_DEFAULT_DASHBOARD_TITLE']:
                        Common.api_patch(
                            f'{api_url}/org/preferences', api_headers,
                            {'homeDashboardUID': dashboard_content['uid']}
                        )

                        print(f' [{dashboard_content["title"]}] set as default dashboard in Grafana')

        except:
            raise Exception('Failed to import Grafana dashboards!')


class Prometheus:
    @staticmethod
    def update_scrape_jobs():
        def get_metrics_list(only_used_in_dashboards=False):
            print(' > Fetching available metrics from Prometheus..')

            metric_names = []


        for env_var in [
            'PROMETHEUS_URL', 'PROMETHEUS_USER', 'PROMETHEUS_PASSWORD',
            'PROMETHEUS_SCRAPE_CONFIGS_FILE',
            'GRAFANA_URL', 'GRAFANA_SERVICE_ACCOUNT_TOKEN',
        ]:
            if not env_var in env or env[env_var] in [None, '']:
                raise Exception(f'{env_var}: missing or invalid environment variable!')


def get_args():
    global arg_func_mappings

    arg_func_mappings = {
        '--create-k0sctl-config': {
            'help': 'Render k0sctl config using config.yaml',
            'func': lambda: CreateCluster.create_k0sctl_config(),
        },
        '--create-destroy-script': {
            'help': 'Create destroy_cluster.sh script for deleting the k0s cluster',
            'func': lambda: CreateCluster.create_destroy_script(),
        },
        '--get-flux-params': {
            'help': 'Create FluxCD bootstrap params using config.yaml',
            'func': lambda: CreateCluster.get_flux_params(),
        },
        '--apply-vm-kubernetes-prerequisites': {
            'help': 'Apply prerequisites for running k0sctl on VMs; Runs scripts/lib/kubernetes_prerequisites.sh',
            'func': lambda: CreateCluster.apply_vm_kubernetes_prerequisites(),
        },
        '--import-grafana-dashboards': {
            'help': 'Import Grafana dashboards from a given directory with sub-directories, acting as dashboard folders and names',
            'func': lambda: Grafana.import_dashboards(),
        },
    }

    parser = argparse.ArgumentParser(description='Common utils')

    for arg in arg_func_mappings:
        parser.add_argument(
            arg, help=arg_func_mappings[arg]['help'], required=False, action='store_true',
        )

    args = [
        f'--{arg[0].replace("_", "-")}' for arg in vars(parser.parse_args()).items()
        if not arg[1] in [None, False]
    ]

    if len(args) == 0:
        parser.print_help()
        exit(1)

    return args


if __name__ == '__main__':
    args = get_args()

    for arg in args:
        arg_func_mappings[arg]['func']()
