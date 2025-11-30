import sys, os, subprocess, json
from datetime import datetime
from zoneinfo import ZoneInfo
from google.cloud import storage


# Usage: python backup.py <service>
# Example: python backup.py --backup-vault

# Required env variables
ENV = {
    'required': [
        'GCP_PROJECT', # Backups destination GCP project
        'GCP_BUCKET', # Backups destination GCP bucket
        'GOOGLE_APPLICATION_CREDENTIALS' # Path to GCP service account JSON key file
    ],
    'vault': {
        'required': [
            'VAULT_KV_NAMES', # Comma-separated list of Vault KV names to backup
            'VAULT_ADDRESS', # Vault hostname and port
            'VAULT_ROOT_TOKEN' # Vault root token
        ],
        'optional': {
            'VAULT_SCHEME': os.getenv('VAULT_SCHEME', 'http') # http/https
        }
    },
    'pg-databases': {
        'required': [
            'DATABASES', # Comma-separated list of databases to backup
            'PGHOST', # PostgreSQL hostname
            'PGPORT', # PostgreSQL port
            'PGUSER', # PostgreSQL user
            'PGPASSWORD', # PostgreSQL password
        ],
        'optional': {}
    }
}

# Script args to services map
ARGS = {
    '--backup-vault': 'vault',
    '--backup-pg-databases': 'pg-databases'
}


class Backups:
    def __init__(self, service: str):
        self.service = service
        self.success = True
        self.params = {}
        self.created = []
        self.dir = '/tmp/backups'

        os.makedirs(self.dir, exist_ok=True)
        self.set_params()


    def log(self, msg: str, warn: bool = False, crash: bool = False):
        if crash:
            print(f'[ X ] {msg}')
            raise SystemExit(1)

        if warn:
            print(f'[ ! ] {msg}')
            return True

        print(f'[ > ] {msg}')


    def get_time(self):
        return datetime.now(
            tz=ZoneInfo('Europe/Sofia')
        ).strftime('%Y-%m-%d-%H:%M:%S')


    def set_params(self):
        def set_param(var_name: str):
            if not os.getenv(var_name):
                self.log(f'{var_name}: Missing environment variable', crash=True)

            if var_name in ['DATABASES', 'VAULT_KV_NAMES']:
                self.params[var_name] = [
                    item.strip()
                    for item in os.getenv(var_name, '').split(',')
                    if item.strip()
                ]

                if len(self.params[var_name]) == 0:
                    self.log(f'{var_name}: Must be a comma-separated list', crash=True)

                return True

            self.params[var_name] = os.getenv(var_name)

        for var in ENV['required']:
            set_param(var)

        for var in ENV[self.service]['required']:
            set_param(var)

        for var in ENV[self.service]['optional']:
            set_param(var)


    def run_cmd(self, cmd: list, **kwargs):
        env = os.environ.copy()

        if self.service == 'vault':
            env.update({
                'VAULT_ADDR': f'{self.params['VAULT_SCHEME']}://{self.params['VAULT_ADDRESS']}',
                'VAULT_TOKEN': self.params['VAULT_ROOT_TOKEN']
            })

        process = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=env,
            **kwargs
        )
        stdout, stderr = process.communicate()

        if process.returncode != 0:
            raise RuntimeError(f'"{" ".join(cmd)}" returned non-zero code: {stderr.decode()}')
        
        return stdout.decode()


    def create_vault_backup(self):
        def export_kv(path_prefix: str, dir: str):
            items = json.loads(
                self.run_cmd(['vault', 'kv', 'list', '-format=json', path_prefix])
            )

            for item in items:
                if item.endswith('/'):
                    child_path = f'{path_prefix}{item}'
                    child_dir = os.path.join(dir, item.strip('/'))

                    os.makedirs(child_dir, exist_ok=True)
                    export_kv(child_path, child_dir)

                else:
                    secret_path = f'{path_prefix}{item}'
                    secret_file = os.path.join(dir, f'{item}.json')
                    secret_data = self.run_cmd(['vault', 'kv', 'get', '-format=json', secret_path])

                    os.makedirs(dir, exist_ok=True)

                    with open(secret_file, 'w') as file:
                        file.write(secret_data)


        for item in self.params['VAULT_KV_NAMES']:
            kv_path = f'{item}/'
            kv_dir = os.path.join(self.dir, item.replace('/', '_'))

            time = self.get_time()
            archive_name = f'{item}@{time}.zip'
            file_path = os.path.join(self.dir, archive_name)

            os.makedirs(kv_dir, exist_ok=True)

            try:
                self.log(f'vault: Exporting secrets from "{item}"..')
                export_kv(kv_path, kv_dir)

                self.log(f'vault: Inflating "{archive_name}"..')
                self.run_cmd(
                    ['zip', '-r', archive_name, '.'],
                    cwd=self.dir
                )

                self.created.append(file_path)

            except Exception as err:
                self.log(f'vault: Failed to backup "{item}", got this -> {err}', warn=True)
                self.success = False


    def create_pg_backup(self):
        for item in self.params['DATABASES']:
            self.log(f'pg: Dumping "{item}"..')
            time = self.get_time()

            try:
                file_path = os.path.join(self.dir, f'{item}@{time}.dump')
                self.run_cmd([
                    'pg_dump',
                    '-h', self.params['PGHOST'],
                    '-p', self.params['PGPORT'],
                    '-U', self.params['PGUSER'],
                    item,
                    '-F', 'c',
                    '-f', file_path
                ])

                self.created.append(file_path)

            except Exception as err:
                self.log(f'pg: Failed to backup "{item}", got this -> {err}', warn=True)
                self.success = False


    def upload_archives(self):
        try:
            client = storage.Client()
            bucket = client.bucket(self.params['GCP_BUCKET'])

        except Exception as err:
            self.log(f'gcp: Unable to connect to Google Cloud Storage, got this -> {err}', crash=True)

        for file_path in self.created:
            try:
                file_name = os.path.basename(file_path)
                file_prefix = file_name.split('@')[0]

                self.log(f'gcp: Uploading "{file_name}" -> GCS/{self.params['GCP_PROJECT']}/{self.params['GCP_BUCKET']}..')

                blob = bucket.blob(file_name)
                blob.upload_from_filename(file_path)

                self.log(f'gcp: Cleaning up old backups..')

                try:
                    for blob in bucket.list_blobs(prefix=f'{file_prefix}@'):
                        if blob.name != file_name:
                            self.log(f'gcp: Deleting "{blob.name}" from GCS/{self.params['GCP_PROJECT']}/{self.params['GCP_BUCKET']}..')
                            blob.delete()

                except Exception as delete_err:
                    self.log(f'gcp: Failed, got this -> {delete_err}', warn=True)

            except Exception as err:
                self.log(f'gcp: Failed to upload "{file_name}", got this -> {err}', warn=True)


if __name__ == '__main__':
    try:
        arg = sys.argv[1]

        assert arg in ['--backup-vault', '--backup-pg-databases']

    except Exception:
        print('Usage: python backup.py <service>')
        print('Example: python backup.py --backup-vault')

        raise SystemExit(1)
    
    backup_service = ARGS[arg]
    backups = Backups(backup_service)

    if backup_service == 'vault':
        backups.create_vault_backup()

    elif backup_service == 'pg-databases':
        backups.create_pg_backup()

    backups.upload_archives()
