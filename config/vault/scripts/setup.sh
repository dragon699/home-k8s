#!/bin/bash


# // Required variables
# VAULT_SCHEME => http/https
# VAULT_ADDRESS => <String> Vault address
# VAULT_ROOT_TOKEN => <String> Vault root token

# // Variables for initial vault setup
# VAULT_USER_EXTERNAL_SECRETS_NAME => <String> Name of the user to create which will be used by External Secrets
# VAULT_USER_EXTERNAL_SECRETS_PASSWORD => <String> Password of the user to create which will be used by External Secrets
# VAULT_POLICY_EXTERNAL_SECRETS_NAME => <String> Name of the policy to create for External Secrets
# VAULT_POLICY_EXTERNAL_SECRETS_FILE => <String> Path to the policy file for External Secrets
# VAULT_KV_NAME => <String> Name of the KV to create (i.e. 'home')


function vault_connect() {
    echo "${VAULT_ADDRESS}: Waiting to start up.."

    until vault status &>/dev/null; do
        echo "${VAULT_ADDRESS}: Still not available, retrying in 5 seconds.."

        sleep 5
    done

    echo "${VAULT_ADDRESS}: Logging in.."

    vault login -method=token token="${VAULT_ROOT_TOKEN}" > /dev/null 2>&1

    [ $? -ne 0 ] && {
        echo "${VAULT_ADDRESS}: Failed!"

        exit 1
    }

    echo "${VAULT_ADDRESS}: Success!"
}

function vault_setup_external_secrets() {
    echo "${VAULT_ADDRESS}: Creating KV store ${VAULT_KV_NAME}.."

    vault secrets enable -path="${VAULT_KV_NAME}" kv-v2 > /dev/null 2>&1

    if [ $? -ne 0 ]; then
        echo "${VAULT_ADDRESS}: Failed or already exists!"

    else
        echo "${VAULT_ADDRESS}: Created!"

    fi

    echo "${VAULT_ADDRESS}: Creating policy ${VAULT_POLICY_EXTERNAL_SECRETS_NAME}.."

    echo "${VAULT_POLICY_EXTERNAL_SECRETS_CONTENT}" | {
        vault policy write "${VAULT_POLICY_EXTERNAL_SECRETS_NAME}" - > /dev/null 2>&1
    }

    if [ $? -ne 0 ]; then
        echo "${VAULT_ADDRESS}: Failed or already exists!"

    else
        echo "${VAULT_ADDRESS}: Created!"

    fi

    echo "${VAULT_ADDRESS}: Enabling authentication method userpass.."

    vault auth enable \
        -path="userpass" \
        -default-lease-ttl=768h \
        -max-lease-ttl=8760h \
        userpass > /dev/null 2>&1

    if [ $? -ne 0 ]; then
        echo "${VAULT_ADDRESS}: Failed or already enabled!"

    else
        echo "${VAULT_ADDRESS}: Enabled!"

    fi

    echo "${VAULT_ADDRESS}: Enabled!"
    echo "${VAULT_ADDRESS}: Updating UI visibility settings for userpass.."

    vault auth tune \
        -listing-visibility=unauth \
        userpass > /dev/null 2>&1

    [ $? -ne 0 ] && {
        echo "${VAULT_ADDRESS}: Failed!"
        exit 1
    }

    echo "${VAULT_ADDRESS}: Updated!"
    echo "${VAULT_ADDRESS}: Creating user ${VAULT_USER_EXTERNAL_SECRETS_NAME}.."

    vault write auth/userpass/users/"${VAULT_USER_EXTERNAL_SECRETS_NAME}" \
        password="${VAULT_USER_EXTERNAL_SECRETS_PASSWORD}" \
        policies="default" > /dev/null 2>&1

    if [ $? -ne 0 ]; then
        echo "${VAULT_ADDRESS}: Failed or already exists!"
    
    else
        echo "${VAULT_ADDRESS}: Created!"

    fi

    echo "${VAULT_ADDRESS}: Created user ${VAULT_USER_EXTERNAL_SECRETS_NAME}!"
}

function main() {
    export VAULT_ADDR="${VAULT_SCHEME}://${VAULT_ADDRESS}"
    export VAULT_TOKEN="${VAULT_ROOT_TOKEN}"

    export VAULT_POLICY_EXTERNAL_SECRETS_CONTENT="$(cat ${VAULT_POLICY_EXTERNAL_SECRETS_FILE})"

    echo "${VAULT_ADDRESS}: Beginning initial setup.."

    vault_connect
    vault_setup_external_secrets

    echo "${VAULT_ADDRESS}: Initial setup completed!"
}

main
