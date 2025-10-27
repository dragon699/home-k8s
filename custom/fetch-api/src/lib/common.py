import json, requests


def read_file(file, as_json=True):
    with open(file, 'r') as f:
        contents = f.read()

    if as_json:
        return json.loads(contents)

    return contents


def make_request(url, method='GET', headers={}, data=None):
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
