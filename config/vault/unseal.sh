#!/bin/bash


# Required variables
# VAULT_SCHEME => http/https
# VAULT_ADDRESSES => String with space-separated Vault addresses
# VAULT_UNSEAL_KEYS => String with space-separated unseal keys


function vault_check_seal_status() {
    echo "${VAULT_ADDR}: Checking seal status.."

    SEAL_STATUS="$(vault status | grep 'Sealed' | awk '{print $2}')"

    if [ "$SEAL_STATUS" = "true" ]; then
        echo "${VAULT_ADDR}: Sealed!"
        vault_unseal

    elif [ "$SEAL_STATUS" = "false" ]; then
        echo "${VAULT_ADDR}: Unsealed - no action needed!"

    else
        echo "${VAULT_ADDR}: Unable to determine seal status!"
        exit 1

    fi
}

function vault_unseal(){
    echo "${VAULT_ADDR}: Unsealing.."

    for KEY in ${VAULT_UNSEAL_KEYS}; do
        echo "${VAULT_ADDR}: Submitting unseal key -> $(echo "$KEY" | cut -c1-3)*****"
        echo "$KEY"
        echo -n "$KEY"

        vault operator unseal "${KEY}"

        if [ $? -ne 0 ]; then
            echo "${VAULT_ADDR}: Vault declined the key -> $(echo "$KEY" | cut -c1-3)*****"

            exit 1
        fi
    done

    echo "${VAULT_ADDR}: Unsealed successfully!"
}

function main() {
    for ADDR in "${VAULT_ADDRESSES}"; do
        export VAULT_ADDR="${VAULT_SCHEME}://${ADDR}"

        vault_check_seal_status
    done
}

main
