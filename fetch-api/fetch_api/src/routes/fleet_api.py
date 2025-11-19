import requests, os
from fastapi import APIRouter, Response, HTTPException



FLEET_API_URL = "https://fleet-api.prd.eu.vn.cloud.tesla.com"
FLEET_AUTH_URL = 'https://fleet-auth.prd.vn.cloud.tesla.com/oauth2/v3/token'
AUDIENCE = FLEET_API_URL

ACCESS_TOKEN_FILE = '/app/fleet-api/access_token'
REFRESH_TOKEN_FILE = '/app/fleet-api/refresh_token'

router = APIRouter()


@router.get('/callback')
def tesla_callback(code: str):
    data = {
        'grant_type': 'authorization_code',
        'client_id': os.getenv('CLIENT_ID'),
        'client_secret': os.getenv('CLIENT_SECRET'),
        'code': code,
        'redirect_uri': 'https://fleet.car.k8s.iaminyourpc.xyz/car/callback',
        'audience': AUDIENCE
    }

    response = requests.post(FLEET_AUTH_URL, data=data)
    if response.status_code != 200:
        raise HTTPException(response.status_code, response.text)

    tokens = response.json()

    with open(ACCESS_TOKEN_FILE, 'w') as f:
        f.write(tokens['access_token'])

    with open(REFRESH_TOKEN_FILE, 'w') as f:
        f.write(tokens['refresh_token'])

    return {'status': 'ok'}


def get_access_token():
    if os.path.exists(ACCESS_TOKEN_FILE):
        return open(ACCESS_TOKEN_FILE).read().strip()
    return refresh_access_token()


def refresh_access_token():
    refresh_token = open(REFRESH_TOKEN_FILE).read().strip()

    data = {
        'grant_type': 'refresh_token',
        'client_id': os.getenv('CLIENT_ID'),
        'refresh_token': refresh_token,
        'audience': AUDIENCE
    }

    resp = requests.post(FLEET_AUTH_URL, data=data)
    if resp.status_code != 200:
        raise HTTPException(resp.status_code, resp.text)

    tokens = resp.json()

    with open(ACCESS_TOKEN_FILE, 'w') as f:
        f.write(tokens['access_token'])

    with open(REFRESH_TOKEN_FILE, 'w') as f:
        f.write(tokens['refresh_token'])

    return tokens['access_token']


def get_partner_token():
    data = {
        'grant_type': 'client_credentials',
        'client_id': os.getenv('CLIENT_ID'),
        'client_secret': os.getenv('CLIENT_SECRET'),
        'audience': FLEET_API_URL
    }

    resp = requests.post(FLEET_AUTH_URL, data=data)
    if resp.status_code != 200:
        raise HTTPException(resp.status_code, resp.text)

    return resp.json()['access_token']


@router.post('/register')
def register_fleet():
    partner_token = get_partner_token()

    url = f'{FLEET_API_URL}/api/1/partner_accounts'

    headers = {
        'Authorization': f'Bearer {partner_token}',
        'X-Tesla-User-Agent': 'fetch-api',
        'Content-Type': 'application/json'
    }

    data = {
        'public_key': 'https://fleet.car.k8s.iaminyourpc.xyz/car/.well-known/appspecific/com.tesla.3p.public-key.pem'
    }

    resp = requests.post(url, headers=headers, json=data)
    if resp.status_code != 200:
        raise HTTPException(resp.status_code, resp.text)

    return resp.json()


@router.get('/list/vehicles')
def list_vehicles():
    access_token = get_access_token()

    url = f'{FLEET_API_URL}/api/1/vehicles'

    headers = {
        'Authorization': f'Bearer {access_token}',
        'X-Tesla-User-Agent': 'fetch-api'
    }

    response = requests.get(url, headers=headers)

    if response.status_code != 200:
        raise HTTPException(response.status_code, response.text)

    return response.json()
