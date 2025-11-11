import json, yaml
import os


def read_file(path: str, type='json'):
    if not os.path.exists(path):
        return None

    with open(path, 'r') as file:
        if type == 'json':
            return json.load(file)

        elif type in ('yaml', 'yml'):
            return yaml.safe_load(file)

        else:
            return file.read()
