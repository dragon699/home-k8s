import sys, os, subprocess, json, shutil, shlex, uuid
import requests
from datetime import datetime
from time import monotonic, sleep
from zoneinfo import ZoneInfo
from google.cloud import storage


# Usage: python backup.py <service> <params>
# Example: python backup.py --backup-vault
#          python backup.py --backup-pg-databases --enable-local-backup
#          python backup.py --backup-longhorn --enable-local-backup --disable-cloud-backup

# Required env variables
ENV = {
    'cloud-backup': {
        'required': [
            'GCP_PROJECT', # Backups destination GCP project
            'GCP_BUCKET', # Backups destination GCP bucket
            'GOOGLE_APPLICATION_CREDENTIALS' # Path to GCP service account JSON key file
        ]
    },
    'encryption': {
        'optional': {
            'AGE_PUBLIC_KEY': os.getenv('AGE_PUBLIC_KEY') # age public key (age1...) - when set, archives are encrypted before upload
        }
    },
    'metrics': {
        'optional': {
            'METRICS_FILE': os.getenv('METRICS_FILE'), # node-exporter textfile to write the backup size metrics to
            'METRICS_CRONJOB': os.getenv('METRICS_CRONJOB') # Name of the CronJob running the backup, used as the metrics label
        }
    },
    'local-backup': {
        'required': [
            'LOCAL_BACKUP_DIR' # Required with --enable-local-backup - directory in which to save backup archives in addition to GCP
        ]
    },
    'vault': {
        'required': [
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
    },
    'longhorn': {
        'required': [
            'LONGHORN_API', # Longhorn manager API URL, e.g. http://longhorn-backend.longhorn.svc:9500
            'LONGHORN_TARGET_DIR' # Mounted Longhorn backup target directory, holding the backupstore folder
        ],
        'optional': {
            'LONGHORN_NAMESPACE': os.getenv('LONGHORN_NAMESPACE', 'longhorn'), # Namespace of the Longhorn CRs
            'LONGHORN_CONCURRENCY': int(os.getenv('LONGHORN_CONCURRENCY', '2')), # Volumes to back up at the same time
            'LONGHORN_TIMEOUT': int(os.getenv('LONGHORN_TIMEOUT', '7200')) # Seconds to wait for a single volume backup
        }
    }
}

# Script args to services map
ARGS = {
    '--backup-vault': 'vault',
    '--backup-pg-databases': 'pg-databases',
    '--backup-longhorn': 'longhorn'
}

# Script flags
FLAGS = [
    '--enable-local-backup',
    '--disable-cloud-backup'
]


class Backups:
    def __init__(self, service: str, local_backups: bool = False, cloud_backups: bool = True):
        self.service = service
        self.local_backups = local_backups
        self.cloud_backups = cloud_backups

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

            if var_name in ['DATABASES']:
                self.params[var_name] = [
                    item.strip()
                    for item in os.getenv(var_name, '').split(',')
                    if item.strip()
                ]

                if len(self.params[var_name]) == 0:
                    self.log(f'{var_name}: Must be a comma-separated list', crash=True)

                return True

            self.params[var_name] = os.getenv(var_name)

        if self.cloud_backups:
            for var in ENV['cloud-backup']['required']:
                set_param(var)

        for var, default in ENV['encryption']['optional'].items():
            self.params[var] = default

        for var, default in ENV['metrics']['optional'].items():
            self.params[var] = default

        if self.params['METRICS_FILE'] and not self.params['METRICS_CRONJOB']:
            self.log('METRICS_CRONJOB: Required with METRICS_FILE', crash=True)

        if self.params['AGE_PUBLIC_KEY'] and not self.params['AGE_PUBLIC_KEY'].startswith('age1'):
            self.log('AGE_PUBLIC_KEY: Must be an age public key (age1...)', crash=True)

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
            try:
                items = json.loads(
                    self.run_cmd(['vault', 'kv', 'list', '-format=json', path_prefix])
                )

            except RuntimeError as err:
                # Empty KV / path
                if 'No value found' in str(err):
                    return True

                raise

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


        try:
            mounts = json.loads(
                self.run_cmd(['vault', 'secrets', 'list', '-format=json'])
            )

        except Exception as err:
            self.log(f'vault: Unable to list secret engines, got this -> {err}', crash=True)

        kv_names = sorted(
            path.strip('/')
            for path, mount in mounts.items()
            if mount.get('type') == 'kv'
        )

        if len(kv_names) == 0:
            self.log('vault: No KV secret engines found', crash=True)

        time = self.get_time()
        archive_name = f'vault-kv@{time}.zip'
        file_path = os.path.join(self.dir, archive_name)
        export_dir = os.path.join(self.dir, 'vault-kv')

        try:
            for item in kv_names:
                kv_dir = os.path.join(export_dir, item.replace('/', '_'))
                os.makedirs(kv_dir, exist_ok=True)

                self.log(f'vault: Exporting secrets from "{item}"..')
                export_kv(f'{item}/', kv_dir)

            self.log(f'vault: Inflating "{archive_name}"..')
            self.run_cmd(
                ['zip', '-r', file_path, os.path.basename(export_dir)],
                cwd=self.dir
            )

            self.created.append(file_path)

        except Exception as err:
            # Skip the archive entirely, so a partial backup never replaces the last full one
            self.log(f'vault: Failed to backup KVs, got this -> {err}', warn=True)
            self.success = False
            self.remove_local(file_path)

        finally:
            self.remove_local(export_dir)


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


    def k8s_request(self, method: str, path: str, **kwargs):
        sa_dir = '/var/run/secrets/kubernetes.io/serviceaccount'

        with open(os.path.join(sa_dir, 'token')) as file:
            token = file.read().strip()

        response = requests.request(
            method,
            f'https://kubernetes.default.svc{path}',
            headers={'Authorization': f'Bearer {token}'},
            verify=os.path.join(sa_dir, 'ca.crt'),
            timeout=30,
            **kwargs
        )

        if method == 'DELETE' and response.status_code == 404:
            return None

        response.raise_for_status()

        return response.json()


    def longhorn_request(self, method: str, path: str, **kwargs):
        response = requests.request(
            method,
            f'{self.params['LONGHORN_API']}{path}',
            timeout=30,
            **kwargs
        )
        response.raise_for_status()

        return response.json()


    def create_longhorn_backup(self):
        crd_path = f'/apis/longhorn.io/v1beta2/namespaces/{self.params['LONGHORN_NAMESPACE']}'
        concurrency = max(self.params['LONGHORN_CONCURRENCY'], 1)

        def list_volume_backups(volume: str):
            return self.k8s_request(
                'GET',
                f'{crd_path}/backups',
                params={'labelSelector': f'backup-volume={volume}'}
            )['items']

        def describe(volume: dict):
            status = volume.get('kubernetesStatus') or {}
            pvc = status.get('pvcName')

            return f'{status.get('namespace')}/{pvc}' if pvc else volume['name']

        def start_backup(volume: dict):
            existing = {item['metadata']['name'] for item in list_volume_backups(volume['name'])}

            snapshot = {'name': f'longhorn-backup-{uuid.uuid4().hex[:12]}', 'labels': {'backup-job': 'longhorn-backup'}}

            self.log(f'longhorn: Backing up "{describe(volume)}"..')

            for action in ('snapshotCreate', 'snapshotBackup'):
                self.longhorn_request(
                    'POST',
                    f'/v1/volumes/{volume['name']}',
                    params={'action': action},
                    json=snapshot
                )

            return existing

        def wait_backup(volume: dict, existing: set):
            deadline = monotonic() + self.params['LONGHORN_TIMEOUT']

            while monotonic() < deadline:
                for item in list_volume_backups(volume['name']):
                    if item['metadata']['name'] in existing:
                        continue

                    if item.get('status', {}).get('state') in ('Completed', 'Error'):
                        return item

                sleep(10)

            return None

        try:
            volumes = self.longhorn_request('GET', '/v1/volumes')['data']

        except Exception as err:
            self.log(f'longhorn: Unable to list volumes, got this -> {err}', crash=True)

        attached = []

        for volume in sorted(volumes, key=lambda item: describe(item)):
            if volume.get('state') == 'attached':
                attached.append(volume)

            else:
                self.log(f'longhorn: Skipping "{describe(volume)}", volume is {volume.get('state')}', warn=True)

        completed = {}

        for index in range(0, len(attached), concurrency):
            batch = attached[index:index + concurrency]
            started = []

            for volume in batch:
                try:
                    started.append((volume, start_backup(volume)))

                except Exception as err:
                    self.log(f'longhorn: Failed to start backup of "{describe(volume)}", got this -> {err}', warn=True)
                    self.success = False

            for volume, existing in started:
                try:
                    backup = wait_backup(volume, existing)

                except Exception as err:
                    self.log(f'longhorn: Lost track of "{describe(volume)}" backup, got this -> {err}', warn=True)
                    self.success = False
                    continue

                if backup is None:
                    self.log(f'longhorn: Backup of "{describe(volume)}" timed out', warn=True)
                    self.success = False

                elif backup['status']['state'] == 'Error':
                    self.log(f'longhorn: Backup of "{describe(volume)}" failed -> {backup['status'].get('error') or backup['status'].get('messages')}', warn=True)
                    self.success = False

                else:
                    self.log(f'longhorn: Backup of "{describe(volume)}" completed')
                    completed[volume['name']] = (volume, backup)

        deleted = []

        for name, (volume, backup) in completed.items():
            for item in list_volume_backups(name):
                if item['metadata']['name'] == backup['metadata']['name']:
                    continue

                snapshot = item.get('status', {}).get('snapshotName') or item['spec'].get('snapshotName')

                try:
                    self.log(f'longhorn: Deleting old backup "{item['metadata']['name']}" of "{describe(volume)}"..')
                    self.k8s_request('DELETE', f'{crd_path}/backups/{item['metadata']['name']}')
                    deleted.append(item['metadata']['name'])

                    if snapshot:
                        self.k8s_request('DELETE', f'{crd_path}/snapshots/{snapshot}')

                except Exception as err:
                    self.log(f'longhorn: Failed to delete old backup "{item['metadata']['name']}", got this -> {err}', warn=True)

        if deleted:
            self.log(f'longhorn: Waiting for {len(deleted)} old backup(s) to be removed from the backupstore..')
            deadline = monotonic() + self.params['LONGHORN_TIMEOUT']

            while monotonic() < deadline:
                remaining = {
                    item['metadata']['name']
                    for item in self.k8s_request('GET', f'{crd_path}/backups')['items']
                } & set(deleted)

                if not remaining:
                    break

                sleep(10)

            else:
                self.log(f'longhorn: Old backups still being removed, archiving anyway', warn=True)

        encrypt = bool(self.params['AGE_PUBLIC_KEY'])
        archive_name = f'longhorn-backupstore@{self.get_time()}.tar'
        file_path = os.path.join(self.dir, f'{archive_name}.age' if encrypt else archive_name)

        tar_cmd = f'tar -C {shlex.quote(self.params['LONGHORN_TARGET_DIR'])} -cf - backupstore'
        cmd = f'{tar_cmd} | age -r {shlex.quote(self.params['AGE_PUBLIC_KEY'])} -o {shlex.quote(file_path)}' if encrypt else f'{tar_cmd} > {shlex.quote(file_path)}'

        try:
            self.log(f'longhorn: Archiving the backupstore into "{os.path.basename(file_path)}"..')
            self.run_cmd(['bash', '-c', f'set -o pipefail; {cmd}'])
            self.created.append(file_path)

        except Exception as err:
            self.log(f'longhorn: Failed to archive the backupstore, got this -> {err}', warn=True)
            self.success = False
            self.remove_local(file_path)


    def save_local(self, file_path: str):
        file_name = os.path.basename(file_path)
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
            old_path = os.path.join(local_dir, old_name)

            # Files only - never touch folders, e.g. a manually extracted backup
            if old_name.startswith(f'{file_prefix}@') and old_name != file_name and os.path.isfile(old_path):
                self.log(f'local: Deleting "{old_name}" from {local_dir}..')
                self.remove_local(old_path)


    def encrypt_archives(self):
        if not self.params['AGE_PUBLIC_KEY']:
            self.log('age: AGE_PUBLIC_KEY not set, archives will NOT be encrypted', warn=True)
            return True

        encrypted = []

        for file_path in self.created:
            file_name = os.path.basename(file_path)
            encrypted_path = f'{file_path}.age'

            if file_path.endswith('.age'):
                encrypted.append(file_path)
                continue

            try:
                self.log(f'age: Encrypting "{file_name}"..')
                self.run_cmd([
                    'age',
                    '-r', self.params['AGE_PUBLIC_KEY'],
                    '-o', encrypted_path,
                    file_path
                ])

                encrypted.append(encrypted_path)

            except Exception as err:
                # Never fall back to uploading the plaintext archive
                self.log(f'age: Failed to encrypt "{file_name}", got this -> {err}', warn=True)
                self.success = False
                self.remove_local(encrypted_path)

            finally:
                self.remove_local(file_path)

        self.created = encrypted


    def measure_archives(self):
        self.size = sum(os.path.getsize(file_path) for file_path in self.created if os.path.exists(file_path))


    def format_size(self, size: float):
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size < 1000:
                return f'{size:.2f} {unit}'

            size /= 1000

        return f'{size:.2f} TB'


    def write_metrics(self):
        metrics_file = self.params['METRICS_FILE']

        if not metrics_file:
            return True

        if not self.success or not self.created:
            self.log('metrics: Run was not fully successful, keeping the previous size metrics', warn=True)
            return False

        labels = f'cronjob="{self.params['METRICS_CRONJOB']}"'
        previous = self.size

        try:
            with open(metrics_file) as file:
                for line in file:
                    if line.startswith('cronjob_backup_size_bytes{'):
                        previous = int(float(line.split()[-1]))

        except FileNotFoundError:
            pass

        lines = [
            '# TYPE cronjob_backup_previous_size_bytes gauge',
            f'cronjob_backup_previous_size_bytes{{{labels}}} {previous}',
            '# TYPE cronjob_backup_size_bytes gauge',
            f'cronjob_backup_size_bytes{{{labels}}} {self.size}',
            '# TYPE cronjob_backup_info gauge',
            f'cronjob_backup_info{{{labels},size="{self.format_size(self.size)}"}} 1'
        ]

        try:
            with open(f'{metrics_file}.tmp', 'w') as file:
                file.write('\n'.join(lines) + '\n')

            os.replace(f'{metrics_file}.tmp', metrics_file)
            self.log(f'metrics: Backup size {self.format_size(self.size)} written to "{metrics_file}"')

        except Exception as err:
            self.log(f'metrics: Failed to write "{metrics_file}", got this -> {err}', warn=True)


    def upload_archives(self):
        bucket = None

        if not self.cloud_backups:
            self.log('gcp: Cloud backup disabled, archives will NOT be uploaded', warn=True)

        else:
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
                if not self.cloud_backups:
                    continue

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
        assert all(flag in FLAGS for flag in flags)

        local_backup = '--enable-local-backup' in flags
        cloud_backup = '--disable-cloud-backup' not in flags
        assert local_backup or cloud_backup

    except Exception:
        print('Usage: python backup.py <service> <params>')
        print('Example: python backup.py --backup-vault')
        print('         python backup.py --backup-pg-databases --enable-local-backup')
        print('         python backup.py --backup-longhorn --enable-local-backup --disable-cloud-backup')

        raise SystemExit(1)

    backup_service = ARGS[service]
    backups = Backups(backup_service, local_backup, cloud_backup)

    if backup_service == 'vault':
        backups.create_vault_backup()

    elif backup_service == 'pg-databases':
        backups.create_pg_backup()

    elif backup_service == 'longhorn':
        backups.create_longhorn_backup()

    backups.encrypt_archives()
    backups.measure_archives()
    backups.upload_archives()
    backups.write_metrics()

    if not backups.success:
        backups.log('Finished with errors, see the warnings above', crash=True)

    backups.log('Have a good day <:')
