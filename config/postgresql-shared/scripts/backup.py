import os, subprocess
from datetime import datetime
from zoneinfo import ZoneInfo
from google.cloud import storage


VARS = [
    'PGHOST',
    'PGPORT',
    'PGUSER',
    'PGPASSWORD',
    'DATABASES',
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
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        stdout, stderr = process.communicate()

        if process.returncode != 0:
            raise RuntimeError(f'"{" ".join(cmd)}" returned non-zero code: {stderr.decode()}')


    def set_vars(self):
        for var in VARS:
            if not os.getenv(var):
                self.log(f'{var}: Missing environment variable', err=True)

            if var == 'DATABASES':
                self.databases = [
                    db.strip()
                    for db in os.getenv(var, '').split(',')
                    if db.strip()
                ]
                continue

            else:
                self.params[var] = os.getenv(var)

        if len(self.databases) == 0:
            self.log('DATABASES: Must be a comma-separated list of database names to backup', err=True)


    def create(self):
        for db in self.databases:
            self.log(f'{db} is being backed up..')

            time = datetime.now(
                tz=ZoneInfo('Europe/Sofia')
            ).strftime('%Y-%m-%dT%H-%M-%S')

            try:
                output_file = f'{self.backups_dir}/{db}@{time}.dump'
                self.run_cmd([
                    'pg_dump',
                    '-h', self.params['PGHOST'],
                    '-p', self.params['PGPORT'],
                    '-U', self.params['PGUSER'],
                    db, '-F', 'c',
                    '-f', output_file
                ])

                self.created_backups.append(output_file)
                self.log(f'Nice.. {db} just got a new backup!')

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
                self.log(f'Uploading {backup} in Google Cloud Storage -> {self.params["GCP_BUCKET"]}..')

                blob = bucket.blob(dest)
                blob.upload_from_filename(backup)

                self.log(f'Done!')

            except Exception as err:
                self.log(f'Failed to upload!', warn=True)
                print(str(err))

                self.success = False


if __name__ == '__main__':
    backups = Backup()
    backups.create()
    backups.upload()

    if not backups.success:
        raise SystemExit(1)

    raise SystemExit(0)
