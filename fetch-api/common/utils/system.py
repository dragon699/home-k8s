import json
import os


def read_file(path: str, type = 'json'):
    if not os.path.exists(path):
        return None

    with open(path, 'r') as file:
        if type == 'json':
            return json.load(file)

        else:
            return file.read()
