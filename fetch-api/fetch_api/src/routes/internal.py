from fetch_api.settings import (settings, connectors)
from fastapi import APIRouter, Response


router = APIRouter()


@router.get('/health')
def health():
    return {
        'healthy': True,
        'connectors': [
            {
                'name': connectors[conn_name].name,
                'healthy': connectors[conn_name].healthy,
                'last_check': connectors[conn_name].health_last_check,
                'next_check': connectors[conn_name].health_next_check,
                'health_endpoint': connectors[conn_name].health_endpoint
            } for conn_name in connectors
        ]
    }


@router.get('/settings')
def read_settings():
    return {'settings': settings.model_dump()}


@router.get('/connectors')
def read_connectors():
    return {'connectors': connectors}


@router.get('/connectors/{conn_name}')
def read_connector(conn_name: str):
    try:
        return connectors[conn_name].model_dump()

    except:
        return {'error': 'Connector not found'}
