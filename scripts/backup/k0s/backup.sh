#!/bin/bash


# Disk path
BACKUP_DISK="/media/martin/Data"

# Backup directory to store backups in
BACKUP_DIR="${BACKUP_DISK}/Backups/home-k8s/k0s"


function get_time() {
    date +'%d/%m at %H:%M:%S'
}

function ensure_disk_mounted() {
    echo " [$(get_time)] > Checking if backup disk is mounted.."

    if ! mountpoint -q "${BACKUP_DISK}"; then
        echo " [$(get_time)] > Backup disk is not mounted, or failed to access!"
        exit 1
    fi
}

function backup() {
    echo " [$(get_time)] > Starting backup.."

    mkdir -p "${BACKUP_DIR}"
    k0s backup --save-path="${BACKUP_DIR}"
    [[ ${?} -ne 0 ]] && echo " [$(get_time)] > Backup failed!" && exit 1

    echo " [$(get_time)] > Backup completed successfully!"
}

function list_backups() {
    ls "${BACKUP_DIR}" 2>/dev/null | grep -E '^k0s_backup_.*\.tar\.gz$'
}

function delete_backup() {
    local BACKUP_NAME="${1}"
    local BACKUP_PATH="${BACKUP_DIR}/${BACKUP_NAME}"

    [[ -f "${BACKUP_PATH}" ]] && rm -f "${BACKUP_PATH}"
}

function main() {
    echo ""
    ensure_disk_mounted

    echo " [$(get_time)] > Checking existing backups.."
    EXISTING_BACKUPS=$(list_backups)

    backup

    echo " [$(get_time)] > Cleaning up previous backups.."
    for BACKUP in ${EXISTING_BACKUPS}; do
        delete_backup "${BACKUP}"
    done

    echo " [$(get_time)] > New backup: $(list_backups | tr '\n' ' ')"
}

main
