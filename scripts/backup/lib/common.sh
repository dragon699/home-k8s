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
