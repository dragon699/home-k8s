#!/bin/bash


SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

source "${LIB_DIR}/common.sh"


# Disk path
BACKUP_DISK="/media/martin/Data"

# Backup directory to store backups in
BACKUP_DIR="${BACKUP_DISK}/Backups/home-k8s/k0s"

# Backup file names (plaintext .tar.gz kept in the regex so old unencrypted backups get cleaned up)
BACKUP_REGEX='^k0s_backup_.*\.tar\.gz(\.age)?$'

# age public key to encrypt backups with - the archive holds every k8s Secret and the cluster PKI
AGE_PUBLIC_KEY="age170llv6tf2j4sgj6s6tvdycutptmqwsyl3ek3w74fjs4jfewqc5qqfqxsvf"


function backup() {
    say "Starting backup.."

    # Plaintext backup stays on the root disk only, never on Data
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "${TMP_DIR}"' EXIT

    mkdir -p "${BACKUP_DIR}"
    k0s backup --save-path="${TMP_DIR}"
    [[ ${?} -ne 0 ]] && fail "Backup failed!"

    for BACKUP_PATH in "${TMP_DIR}"/k0s_backup_*.tar.gz; do
        BACKUP_NAME="$(basename "${BACKUP_PATH}")"

        say "Encrypting ${BACKUP_NAME}.."
        age -r "${AGE_PUBLIC_KEY}" -o "${BACKUP_DIR}/${BACKUP_NAME}.age" "${BACKUP_PATH}"
        [[ ${?} -ne 0 ]] && rm -f "${BACKUP_DIR}/${BACKUP_NAME}.age" && fail "Encryption failed!"
    done

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
