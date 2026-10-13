#!/bin/sh


# Runs inside a pod, so all variables commented out;
# so they don't erase the ones from the pod runtime;

# VAULT_ADDR => Vault address
# VAULT_ADDR="http://vault.vault.svc:8200"

# VAULT_TOKEN => Token allowed to read sys/storage/raft/snapshot
# VAULT_TOKEN=""

# BACKUP_DIR => Directory to store snapshots in
# BACKUP_DIR=""

# METRICS_FILE => node-exporter textfile to write the backup size metrics to (optional)
# METRICS_FILE=""


SCRIPT_DIR="$(cd "$(dirname "${0}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

. "${LIB_DIR}/common.sh"


# Backup file names
BACKUP_REGEX='^vault-raft_.*\.snap$'


function backup() {
    BACKUP_PATH="${BACKUP_DIR}/vault-raft_$(date +'%Y-%m-%dT%H_%M_%S').snap"
    say "Saving raft snapshot to ${BACKUP_PATH}.."

    vault operator raft snapshot save "${BACKUP_PATH}"
    [ ${?} -ne 0 ] && fail "Snapshot failed!"

    say "Snapshot completed successfully!"
}

function main() {
    say "Checking existing backups.."
    EXISTING_BACKUPS=$(list_backups "${BACKUP_DIR}" "${BACKUP_REGEX}")

    backup

    say "Cleaning up previous backups.."
    for BACKUP in ${EXISTING_BACKUPS}; do
        delete_backup "${BACKUP_DIR}" "${BACKUP}"
    done

    say "New backup: $(list_backups "${BACKUP_DIR}" "${BACKUP_REGEX}" | tr '\n' ' ')"

    if [ -n "${METRICS_FILE}" ]; then
        write_backup_size_metrics "${METRICS_FILE}" vault-backup-full "$(stat -c %s "${BACKUP_PATH}")"
    fi
}

main
