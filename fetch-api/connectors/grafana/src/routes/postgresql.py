from connectors.grafana.src.api_processor import APIProcessor
from fastapi import APIRouter


router = APIRouter()


@router.get('/car-battery')
def get_car_battery():
    return APIProcessor.process_request(
        query_ds_type='postgresql',
        query_id='teslamate-usable-battery-level'
    )


@router.get('/car-last-location')
def get_car_last_location():
    return APIProcessor.process_request(
        query_ds_type='postgresql',
        query_id='teslamate-last-seen-location'
    )


@router.get('/car-state')
def get_car_state():
    return APIProcessor.process_request(
        query_ds_type='postgresql',
        query_id='teslamate-car-state'
    )


@router.get('/car-average-consumption')
def get_car_average_consumption():
    return APIProcessor.process_request(
        query_ds_type='postgresql',
        query_id='teslamate-average-consumption'
    )
