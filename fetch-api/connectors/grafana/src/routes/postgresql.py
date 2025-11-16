from connectors.grafana.src.api_processor import APIProcessor
from fastapi import APIRouter, Query


router = APIRouter()


@router.get('/car-battery')
def get_car_battery():
    return APIProcessor.process_request(
        query_ds_type='postgresql',
        query_id='teslamate-usable-battery-level'
    )


@router.get('/car-last-charge')
def get_car_last_charge():
    return APIProcessor.process_request(
        query_ds_type='postgresql',
        query_id='teslamate-last-charge-info'
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


@router.get('/car-efficiency')
def get_car_average_consumption():
    return APIProcessor.process_request(
        query_ds_type='postgresql',
        query_id='teslamate-car-efficiency'
    )


@router.get('/car-drives-history')
def get_car_drives(number_of_drives: int = Query(5, ge=1, le=1000)):
    return APIProcessor.process_request(
        query_ds_type='postgresql',
        query_id='teslamate-car-drives-info',
        query_params={
            'number_of_drives': number_of_drives
        }
    )
