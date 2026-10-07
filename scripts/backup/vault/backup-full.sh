#!/bin/sh


# // Required variables
# VAULT_ADDR => Vault address, e.g. http://vault.vault.svc:8200
# VAULT_TOKEN => Token allowed to read sys/storage/raft/snapshot
# BACKUP_DIR => Directory to store snapshots in


function get_time() {
    date +'%d/%m at %H:%M:%S'
}

function list_backups() {
    ls "${BACKUP_DIR}" 2>/dev/null | grep -E '^vault-raft_.*\.snap$'
}

function delete_backup() {
    local BACKUP_PATH="${BACKUP_DIR}/${1}"

    [ -f "${BACKUP_PATH}" ] && rm -f "${BACKUP_PATH}"
}

function backup() {
    BACKUP_PATH="${BACKUP_DIR}/vault-raft_$(date +'%Y-%m-%dT%H_%M_%S').snap"
    echo " [$(get_time)] > Saving raft snapshot to ${BACKUP_PATH}.."

    vault operator raft snapshot save "${BACKUP_PATH}"
    [ ${?} -ne 0 ] && echo " [$(get_time)] > Snapshot failed!" && exit 1

    echo " [$(get_time)] > Snapshot completed successfully!"
}

function main() {
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
