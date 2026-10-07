#!/bin/bash


SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

source "${LIB_DIR}/common.sh"


# Immich disk path
IMMICH_DISK="/media/martin/Data"

# Immich directory which to backup
IMMICH_DIR="${IMMICH_DISK}/Photos/immich"

# Disk path
BACKUP_DISK="/"

# Backup directory to store backups in
BACKUP_DIR="/srv/Backups/immich"


function backup() {
    say "Starting backup of ${IMMICH_DIR} to ${BACKUP_DIR}.."

    mkdir -p "${BACKUP_DIR}"

    ionice -c3 nice -n19 rsync -a \
        --delete \
        --exclude 'thumbs/' \
        --exclude 'encoded-video/' \
        --info=stats1 \
        "${IMMICH_DIR}/" "${BACKUP_DIR}/"
    [[ ${?} -ne 0 ]] && fail "Backup failed!"

    say "Backup completed successfully!"
}

function main() {
    echo ""
    ensure_disk_mounted "${IMMICH_DISK}"
    ensure_disk_mounted "${BACKUP_DISK}"
    ensure_dir_not_empty "${IMMICH_DIR}/library"

    backup

    say "New backup size: $(du -sh "${BACKUP_DIR}" | cut -f1)"
}

main
