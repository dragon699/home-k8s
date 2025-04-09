#!/bin/bash

# This script aims to optimize Prometheus metrics storage by blacklisting unused metrics;
#
# - Checks all metric names from incoming scrape jobs in Prometheus;
# - Checks which metric names are actively being used in Grafana dashboards;
# - Based on above, all metric names that are not being used in Grafana dashboards
#   will be blacklisted in Prometheus scrape jobs, so they're not kept at all;
# - PROMETHEUS_SCRAPE_CONFIGS_FILE should point to a Prometheus config file containing scrape jobs,
#   or to a Flux HelmRelease kind kubernetes object file containing Prometheus scrape jobs in its values;
# - The changes then need to be commited to the git repository, in order
#   to allow Flux to apply the changes, using the following commands;
#   git pull
#   git add *prometheus.yaml*
#   git commit -m "prometheus: Update list of disabled metrics"
#   git push
#
# - Prerequisites
#   Ensure you have a Grafana service account key, generated from;
#   https://grafana.monitor.procreditbank.bg:30443/org/serviceaccounts
#
# - Edit the configurable parameters below, if needed;
#   IMPORTANT: For URLS, use http, as mimirtool does not support untrusted certificates;
#   Ensure PROMETHEUS_URL, PROMETHEUS_USER and PROMETHEUS_PASSWORD are set correctly;
#   Ensure PROMETHEUS_SCRAPE_CONFIGS_FILE is set correctly;
#   Ensure GRAFANA_URL and GRAFANA_SERVICE_ACCOUNT_TOKEN are set correctly;
#   If Prometheus does not need basic auth, set PROMETHEUS_USER and PROMETHEUS_PASSWORD to empty strings;
# - Run ./sync_unused_metrics.sh to sync the unused metrics in Prometheus scrape jobs;
#   Ensure the changes are commited after;


SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

source "${LIB_DIR}/common.sh"


# Configurable parameters
# Prometheus front-end URL
PROMETHEUS_URL="http://prometheus.monitor.procreditbank.bg:30080"
# If using basic auth: Prometheus username
PROMETHEUS_USER="sys"
# If using basic auth: Prometheus password
PROMETHEUS_PASSWORD=""
# Prometheus scrape configs file
# OR a Flux helm release file for Prometheus;
# with `scrape_configs` in the `values.serverFiles.prometheus.yml.scrape_configs` key;
PROMETHEUS_SCRAPE_CONFIGS_FILE="${LIB_DIR}/../../kubernetes/resources/prod/03 - event-tracking/prometheus/prometheus.yaml"
# Grafana front-end URL
GRAFANA_URL="https://grafana.monitor.procreditbank.bg:30443"
# Grafana service account key
GRAFANA_SERVICE_ACCOUNT_TOKEN=""


function set_env {
    export PROMETHEUS_URL
    export PROMETHEUS_USER
    export PROMETHEUS_PASSWORD
    export PROMETHEUS_SCRAPE_CONFIGS_FILE
    export GRAFANA_URL
    export GRAFANA_SERVICE_ACCOUNT_TOKEN
}


function sync_unused_metrics {
    set_env

    say "Syncing unused metrics in Prometheus scrape jobs..."
    python3 "${LIB_DIR}/utils.py" --sync-unused-prometheus-metrics
}
