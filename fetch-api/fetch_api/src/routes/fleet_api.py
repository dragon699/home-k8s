from fastapi import APIRouter, Response


router = APIRouter()



@router.get('/.well-known/appspecific/com.tesla.3p.public-key.pem')
def get_public_key():
    with open('/app/fleet_api/public/public-key.pem', 'rb') as f:
        key = f.read()
    
    return Response(content=key, media_type='application/x-pem-file')
