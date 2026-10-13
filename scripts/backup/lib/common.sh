#!/bin/sh

# Shared by host scripts (bash) and the vault backup pod (busybox sh);
# keep it POSIX sh compatible: no [[ ]], arrays or BASH_SOURCE;


function get_time() {
    date +'%d/%m at %H:%M:%S'
}

function say() {
    echo " [$(get_time)] > ${1}"
}

function fail() {
    say "${1}"
    exit 1
}

function on_exit() {
    EXIT_HOOKS="${EXIT_HOOKS:+${EXIT_HOOKS}; }${1}"
    trap 'EXIT_CODE=${?}; eval "${EXIT_HOOKS}"' EXIT
}

function log_to_file() {
    [ -t 1 ] && return 0

    mkdir -p "$(dirname "${1}")"
    exec >> "${1}" 2>&1
}

function ensure_disk_mounted() {
    say "Checking if disk ${1} is mounted.."

    if ! mountpoint -q "${1}"; then
        fail "Disk ${1} is not mounted, or failed to access!"
    fi
}

function ensure_dir_not_empty() {
    if [ -z "$(ls -A "${1}" 2>/dev/null)" ]; then
        fail "${1} is empty or missing, refusing to continue!"
    fi
}

function get_dir_size() {
    local SIZE="$(du -sb "${1}" 2>/dev/null | cut -f1)"

    echo "${SIZE:-0}"
}

function format_size() {
    awk -v BYTES="${1}" 'BEGIN {
        split("B KB MB GB TB", UNITS, " ")
        UNIT = 1

        while (BYTES >= 1000 && UNIT < 5) {
            BYTES /= 1000
            UNIT++
        }

        printf "%.2f %s\n", BYTES, UNITS[UNIT]
    }'
}

function write_backup_size_metrics() {
    local METRICS_FILE="${1}"
    local LABELS="cronjob=\"${2}\""
    local SIZE="${3}"
    local PREVIOUS_SIZE="$(awk 'index($1, "cronjob_backup_size_bytes{") == 1 {print $2}' "${METRICS_FILE}" 2>/dev/null)"

    {
        echo "# TYPE cronjob_backup_previous_size_bytes gauge"
        echo "cronjob_backup_previous_size_bytes{${LABELS}} ${PREVIOUS_SIZE:-${SIZE}}"
        echo "# TYPE cronjob_backup_size_bytes gauge"
        echo "cronjob_backup_size_bytes{${LABELS}} ${SIZE}"
        echo "# TYPE cronjob_backup_info gauge"
        echo "cronjob_backup_info{${LABELS},size=\"$(format_size "${SIZE}")\"} 1"
    } > "${METRICS_FILE}.tmp"

    mv "${METRICS_FILE}.tmp" "${METRICS_FILE}"
}

function list_backups() {
    ls "${1}" 2>/dev/null | grep -E "${2}"
}

function delete_backup() {
    local BACKUP_PATH="${1}/${2}"

    [ -f "${BACKUP_PATH}" ] && rm -f "${BACKUP_PATH}"
}

function install_cron() {
    echo "Installing cron job ${2}.."

    sudo install -o root -g root -m 644 "${1}" "/etc/cron.d/${2}"
    [ ${?} -ne 0 ] && echo "Failed to install cron job!" && exit 1

    echo "Cron job installed successfully!"
}
