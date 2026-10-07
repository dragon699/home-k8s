import sys, os, subprocess, json, shutil
from datetime import datetime
from zoneinfo import ZoneInfo
from google.cloud import storage


# Usage: python backup.py <service> <params>
# Example: python backup.py --backup-vault
#          python backup.py --backup-pg-databases --enable-local-backup

# Required env variables
ENV = {
    'required': [
        'GCP_PROJECT', # Backups destination GCP project
        'GCP_BUCKET', # Backups destination GCP bucket
        'GOOGLE_APPLICATION_CREDENTIALS' # Path to GCP service account JSON key file
    ],
    'local-backup': {
        'required': [
            'LOCAL_BACKUP_DIR' # Required with --enable-local-backup - directory in which to save backup archives in addition to GCP
        ]
    },
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
    def __init__(self, service: str, local_backups: bool = False):
        self.service = service
        self.local_backups = local_backups

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

        for var, default in ENV[self.service]['optional'].items():
            self.params[var] = default

        if self.local_backups:
            for var in ENV['local-backup']['required']:
                set_param(var)

            if not os.path.isdir(self.params['LOCAL_BACKUP_DIR']):
                self.log(f'LOCAL_BACKUP_DIR: "{self.params['LOCAL_BACKUP_DIR']}" is not a directory', crash=True)


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


    def remove_local(self, path: str):
        try:
            if os.path.isdir(path):
                shutil.rmtree(path)

            elif os.path.exists(path):
                os.remove(path)

        except Exception as err:
            self.log(f'Failed to remove "{path}", got this -> {err}', warn=True)


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
                    ['zip', '-r', file_path, os.path.basename(kv_dir)],
                    cwd=self.dir
                )

                self.created.append(file_path)

            except Exception as err:
                self.log(f'vault: Failed to backup "{item}", got this -> {err}', warn=True)
                self.success = False
                self.remove_local(file_path)

            finally:
                self.remove_local(kv_dir)


    def create_pg_backup(self):
        for item in self.params['DATABASES']:
            self.log(f'pg: Dumping "{item}"..')
            time = self.get_time()

            file_path = os.path.join(self.dir, f'{item}@{time}.dump')

            try:
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
                self.remove_local(file_path)


    def save_local(self, file_path: str):
        file_name = os.path.basename(file_path)

        if self.service == 'vault':
            file_name = f'vault-kv@{file_name.split('@', 1)[1]}'

        file_prefix = file_name.split('@')[0]
        local_dir = self.params['LOCAL_BACKUP_DIR']

        try:
            self.log(f'local: Saving "{file_name}" -> {local_dir}..')
            shutil.copy2(file_path, os.path.join(local_dir, file_name))

        except Exception as err:
            self.log(f'local: Failed to save "{file_name}", got this -> {err}', warn=True)
            self.success = False
            return False

        self.log(f'local: Cleaning up old backups..')

        for old_name in os.listdir(local_dir):
            if old_name.startswith(f'{file_prefix}@') and old_name != file_name:
                self.log(f'local: Deleting "{old_name}" from {local_dir}..')
                self.remove_local(os.path.join(local_dir, old_name))


    def upload_archives(self):
        bucket = None

        try:
            client = storage.Client()
            bucket = client.bucket(self.params['GCP_BUCKET'])

        except Exception as err:
            self.log(f'gcp: Unable to connect to Google Cloud Storage, got this -> {err}', warn=True)
            self.success = False

        for file_path in self.created:
            file_name = os.path.basename(file_path)
            file_prefix = file_name.split('@')[0]

            try:
                if bucket is None:
                    raise RuntimeError('No connection to Google Cloud Storage')

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
                self.success = False

            finally:
                if self.local_backups:
                    self.save_local(file_path)

                self.remove_local(file_path)


if __name__ == '__main__':
    try:
        service = sys.argv[1]
        assert service in ARGS

        flags = sys.argv[2:]
        assert all(flag == '--enable-local-backup' for flag in flags)

        local_backup = '--enable-local-backup' in flags

    except Exception:
        print('Usage: python backup.py <service> <params>')
        print('Example: python backup.py --backup-vault')
        print('         python backup.py --backup-pg-databases --enable-local-backup')

        raise SystemExit(1)

    backup_service = ARGS[service]
    backups = Backups(backup_service, local_backup)

    if backup_service == 'vault':
        backups.create_vault_backup()

    elif backup_service == 'pg-databases':
        backups.create_pg_backup()

    backups.upload_archives()

    if not backups.success:
        backups.log('Finished with errors, see the warnings above', crash=True)

    backups.log('Have a good day <:')
