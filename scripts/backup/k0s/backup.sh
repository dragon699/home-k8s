#!/bin/bash


SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

source "${LIB_DIR}/common.sh"


# Disk path
BACKUP_DISK="/media/martin/Data"

# Backup directory to store backups in
BACKUP_DIR="${BACKUP_DISK}/Backups/home-k8s/k0s"

# Backup file names
BACKUP_REGEX='^k0s_backup_.*\.tar\.gz$'


function backup() {
    say "Starting backup.."

    mkdir -p "${BACKUP_DIR}"
    k0s backup --save-path="${BACKUP_DIR}"
    [[ ${?} -ne 0 ]] && fail "Backup failed!"

    say "Backup completed successfully!"
}

function main() {
    echo ""
    ensure_disk_mounted "${BACKUP_DISK}"

    say "Checking existing backups.."
    EXISTING_BACKUPS=$(list_backups "${BACKUP_DIR}" "${BACKUP_REGEX}")

    backup

    say "Cleaning up previous backups.."
    for BACKUP in ${EXISTING_BACKUPS}; do
        delete_backup "${BACKUP_DIR}" "${BACKUP}"
    done

    say "New backup: $(list_backups "${BACKUP_DIR}" "${BACKUP_REGEX}" | tr '\n' ' ')"
}

main
