
#!/bin/bash


SLEEP_TIME=6

CONF_DIR="/etc/containerd/conf.d"
CONF_NAME="99-nvidia.toml"
DESIRED_VERSION="2"

K0S_SERVICE="k0scontroller.service"


function list_configs() {
    ls ${CONF_DIR}
}

function restart_k0s() {
    systemctl stop ${K0S_SERVICE}
    systemctl start ${K0S_SERVICE}
}

function fix_version() {
    CONFIG_FILES=$(list_configs)
    CONF_PATH="${CONF_DIR}/${CONF_NAME}"

    if echo ${CONFIG_FILES} | grep -q "${CONF_NAME}"; then
        VERSION=$(head -1 ${CONF_PATH} | grep -E "^version = [0-9]$")

        if [[ ${VERSION} == "3" ]]; then
            sed -i "s/^version = 3/version = ${DESIRED_VERSION}/" ${CONF_PATH}
            restart_k0s

            return "fixed"

        else
            return "ok"

        fi

    else
        return "missing_config"

    fi
}

function main() {
    FIXED=false

    while FIXED == false; do
        RESULT=$(fix_version)

        if echo ${RESULT} | grep -q "missing_config"; then
            sleep ${SLEEP_TIME}

            continue
        fi

        FIXED=true
    done
}


main
