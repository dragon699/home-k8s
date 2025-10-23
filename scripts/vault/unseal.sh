#!/bin/bash


VAULT_ADDRESSES=()
VAULT_UNSEAL_KEYS=()


function vault_check_seal_status() {
    echo "${VAULT_ADDR}: Checking seal status.."

    SEAL_STATUS=$(vault status | grep 'Sealed' | awk '{print $2}')

    if [ "$SEAL_STATUS" = "true" ]; then
        echo "${VAULT_ADDR}: Sealed!"
        vault_unseal

    else
        echo "${VAULT_ADDR}: Unsealed - no action needed!"

    fi
}

function vault_unseal(){
    echo "${VAULT_ADDR}: Unsealing.."

    for KEY in "${VAULT_UNSEAL_KEYS[@]}"; do
        echo "${VAULT_ADDR}: Submitting unseal key -> ${KEY:0:3}*****"

        vault operator unseal "${KEY}"

        if [ $? -ne 0 ]; then
            echo "${VAULT_ADDR}: Vault declined the key -> ${KEY:0:3}*****"

            exit 1
        fi
    done

    echo "${VAULT_ADDR}: Unsealed successfully!"
}

function main() {
    for ADDR in "${VAULT_ADDRESSES[@]}"; do
        export VAULT_ADDR=${ADDR}

        vault_check_seal_status
    done
}

main
