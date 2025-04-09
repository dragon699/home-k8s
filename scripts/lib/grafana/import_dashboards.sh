#!/bin/bash

# - Edit the configurable parameters below, if needed;
#   Ensure GRAFANA_URL, GRAFANA_USER and GRAFANA_PASSWORD are set correctly;
#   Set GRAFANA_DASHBOARDS_DIR to a directory containing Grafana dashboard JSON files;
#   Set GRAFANA_DEFAULT_DASHBOARD_TITLE to the title of the default dashboard to be set;
#   The title could be found in the "title" key of the JSON;
# - Run: ./import_dashboards.sh to import the dashboards;


SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

source "${LIB_DIR}/common.sh"


# Configurable parameters
GRAFANA_URL="https://grafana.monitor.procreditbank.bg:30443"
GRAFANA_USER="martin.v.stanchev"
GRAFANA_PASSWORD=""
GRAFANA_DASHBOARDS_DIR="${SCRIPT_DIR}/dashboards"
GRAFANA_DEFAULT_DASHBOARD_TITLE="API Monitoring"


function set_env {
    export GRAFANA_URL
    export GRAFANA_USER
    export GRAFANA_PASSWORD
    export GRAFANA_DASHBOARDS_DIR
    export GRAFANA_DEFAULT_DASHBOARD_TITLE
}


function import_dashboards {
    set_env

    say "Importing dashboards from ${GRAFANA_DASHBOARDS_DIR}..."
    python3 "${LIB_DIR}/utils.py" --import-grafana-dashboards

    ! [[ $? == 0 ]] && fail "Importing dashboards into Grafana failed!"

    say "Dashboards imported successfully."
}


import_dashboards
