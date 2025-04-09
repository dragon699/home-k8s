#!/bin/bash

# Common helper functions utilized by other bash scripts;


LIB_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"


function say {
    echo -e " [ \033[0;34m>\033[0m ] $1"

    if ! [[ -z $2 ]]; then
        [[ $2 == 'nl' ]] && echo ''
    fi
}


function say_cmd {
    echo -e " [ \033[0;33m$\033[0m ] $1"
}


function fail {
    echo -e " [ \033[0;31m!\033[0m ] $1"
    exit 1
}


function ensure_root {
    if [[ "$(id -u)" != "0" ]]; then
        fail "You must be root!"
    fi
}


function update_bashrc {
    if [[ -z $(grep "$1" ${HOME}/.bashrc) ]]; then
        echo "$1" >> ${HOME}/.bashrc
    fi
}
