#!/bin/bash


SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

source "${LIB_DIR}/common.sh"


# Disk path holding the Longhorn backup target
BACKUP_DISK="/media/martin/Data"

# nfs-server has RequiresMountsFor=${BACKUP_DISK}; when the disk fails to mount at boot
# (e.g. NTFS left dirty by Windows) it ends with "dependency failed" and is never retried,
# even after the disk gets mounted later - so start it once the disk is back


function main() {
    systemctl is-active -q nfs-server && return 0

    if ! mountpoint -q "${BACKUP_DISK}"; then
        say "${BACKUP_DISK} is not mounted, nfs-server stays down - Longhorn backups are failing!"
        return 1
    fi

    say "${BACKUP_DISK} is mounted but nfs-server is down, starting it.."
    systemctl start nfs-server
    [[ ${?} -ne 0 ]] && fail "Failed to start nfs-server!"

    say "nfs-server started!"
}

main
