import os, subprocess, json
from time import sleep
from datetime import datetime
from zoneinfo import ZoneInfo
from google.cloud import storage


VARS = [
    'VAULT_KV_NAMES',
    'VAULT_SCHEME',
    'VAULT_ADDRESS',
    'VAULT_ROOT_TOKEN',
    'GCP_PROJECT',
    'GCP_BUCKET'
]


class Backup:
    def __init__(self):
        self.success = True
        self.params = {}
        self.databases = []
        self.created_backups = []
        self.backups_dir = '/tmp/backups'

        os.makedirs(self.backups_dir, exist_ok=True)
        self.set_vars()


    def log(self, msg: str, err: bool = False, warn: bool = False):
        if err:
            print(f'[ X ] {msg}')
            raise SystemExit(1)

        if warn:
            print(f'[ ! ] {msg}')
            return True

        print(f'[ > ] {msg}')


    def run_cmd(self, cmd: list):
        env = os.environ.copy()
        env.update({
            'VAULT_ADDR': f'{self.params["VAULT_SCHEME"]}://{self.params["VAULT_ADDRESS"]}',
            'VAULT_TOKEN': self.params['VAULT_ROOT_TOKEN']
        })

        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
        stdout, stderr = process.communicate()

        if process.returncode != 0:
            raise RuntimeError(f'"{" ".join(cmd)}" returned non-zero code: {stderr.decode()}')

        return stdout.decode()


    def set_vars(self):
        for var in VARS:
            if not os.getenv(var):
                self.log(f'{var}: Missing environment variable', err=True)

            if var == 'VAULT_KV_NAMES':
                self.kv_names = [
                    kv.strip()
                    for kv in os.getenv(var, '').split(',')
                    if kv.strip()
                ]
                continue

            else:
                self.params[var] = os.getenv(var)

        if len(self.kv_names) == 0:
            self.log('VAULT_KV_NAMES: Must be a comma-separated list of KV paths to backup', err=True)


    def create(self):
        for kv in self.kv_names:
            kv_path = f'{kv}/'
            kv_dir = os.path.join(self.backups_dir, kv)
            os.makedirs(kv_dir, exist_ok=True)

            try:
                self.log(f'{kv_path} is being exported..')
                secrets = json.loads(
                    self.run_cmd(['vault', 'kv', 'list', '-format=json', kv_path])
                )

                for secret in secrets:
                    secret_data = self.run_cmd(['vault', 'kv', 'get', '-format=json', f'{kv_path}{secret}'])
                    secret_dir = os.path.join(kv_dir, secret)
                    secret_file = os.path.join(secret_dir, f'{secret}.json')

                    os.makedirs(secret_dir, exist_ok=True)

                    with open(secret_file, 'w') as file:
                        file.write(secret_data)

                self.created_backups.append(kv_dir)
                self.log(f'Nice, {kv} just got a new backup!')


            except Exception as err:
                self.log(f'Backup failed!', warn=True)
                print(str(err))

                self.success = False


    def upload(self):
        try:
            self.log('Connecting to Google Cloud Storage..')

            client = storage.Client()
            bucket = client.bucket(self.params['GCP_BUCKET'])

        except Exception as err:
            print(str(err))
            self.log(f'Connection failed!', err=True)

        for backup in self.created_backups:
            try:
                dest = backup.split('/')[-1]
                db_name = dest.split('@')[0]

                self.log(f'Uploading {dest} in Google Cloud Storage -> {self.params["GCP_BUCKET"]}..')

                blob = bucket.blob(dest)
                blob.upload_from_filename(backup)

                self.log(f'Done!')

                try:
                    for old_blob in bucket.list_blobs(prefix=f'{db_name}@'):
                        if old_blob.name != dest:
                            self.log(f'Deleting {old_blob.name}..')
                            old_blob.delete()

                    self.log('We are clean now ((:')

                except Exception as delete_err:
                    self.log(f'Failed to delete older backups for {db_name}', warn=True)
                    print(str(delete_err))

            except Exception as err:
                self.log(f'Failed to upload!', warn=True)
                print(str(err))

                self.success = False


if __name__ == '__main__':
    backups = Backup()
    backups.create()
    sleep(3500)

    # if not backups.success:
    #     raise SystemExit(1)

    # raise SystemExit(0)
