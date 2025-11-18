import requests
from fastapi import APIRouter, Response, HTTPException



FLEET_API_URL = 'https://fleet-api.prd.eu.vn.cloud.tesla.com'
TOKEN_URL = 'https://auth.tesla.com/oauth2/v3/token'

ACCESS_TOKEN_FILE = '/app/fleet-api/access_token'
REFRESH_TOKEN_FILE = '/app/fleet-api/refresh_token'


router = APIRouter()


@router.get('/callback')
def tesla_callback(code: str):
    data = {
        'grant_type': 'authorization_code',
        'client_id': '<YOUR CLIENT ID>',
        'client_secret': '<YOUR CLIENT SECRET>',
        'code': code,
        'redirect_uri': 'https://fleet.car.k8s.iaminyourpc.xyz/tesla/callback'
    }

    response = requests.post(TOKEN_URL, data=data)
    if response.status_code != 200:
        raise HTTPException(response.status_code, response.text)

    tokens = response.json()

    access_token = tokens['access_token']
    refresh_token = tokens['refresh_token']

    with open(ACCESS_TOKEN_FILE, 'w') as f:
        f.write(access_token)

    with open(REFRESH_TOKEN_FILE, 'w') as f:
        f.write(refresh_token)

    return {'status': 'ok'}


@router.get('/.well-known/appspecific/com.tesla.3p.public-key.pem')
def get_public_key():
    with open('/app/fleet-api/public/public-key.pem', 'rb') as f:
        key = f.read()
    
    return Response(content=key, media_type='application/x-pem-file')


def get_access_token():
    try:
        acc = open(ACCESS_TOKEN_FILE).read().strip()
        return acc
    except:
        return refresh_access_token()
    

def refresh_access_token():
    refresh_token = open(REFRESH_TOKEN_FILE).read().strip()

    data = {
        'grant_type': 'refresh_token',
        'client_id': '<YOUR CLIENT ID>',
        'client_secret': '<YOUR CLIENT SECRET>',
        'refresh_token': refresh_token
    }

    resp = requests.post(TOKEN_URL, data=data)
    tokens = resp.json()

    new_access = tokens.get('access_token')
    new_refresh = tokens.get('refresh_token')

    with open(ACCESS_TOKEN_FILE, 'w') as f:
        f.write(new_access)

    with open(REFRESH_TOKEN_FILE, 'w') as f:
        f.write(new_refresh)

    return new_access


@router.get('/list/vehicles')
def list_vehicles():
    access_token = get_access_token()

    url = f'{FLEET_API_URL}/api/1/users/me/vehicles'

    headers = {
        'Authorization': f'Bearer {access_token}',
        'X-Tesla-User-Agent': 'fetch-api'
    }

    response = requests.get(url, headers=headers)

    if response.status_code != 200:
        raise HTTPException(response.status_code, response.text)

    return response.json()
