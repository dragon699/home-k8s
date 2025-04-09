#!/bin/bash

# Common install functions utilized by create_cluster.sh;
# Used to install dependency tools for k0s, kubernetes and CD (Flux);


source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)/common.sh"


function install_bash_utils {
    if [[ -z $(which yq) ]]; then
        URL="https://github.com/mikefarah/yq/releases/download/v4.13.4/yq_linux_amd64"

        say "Installing ${URL}.."

        wget ${URL} -O /usr/bin/yq > /dev/null 2>&1
        chmod +x /usr/bin/yq

        say "Installation completed!"
    fi
}


function install_python_dependencies {
    PYTHON_PACKAGES=(
        yaml
        jinja2
        paramiko
    )

    for pkg in ${PYTHON_PACKAGES[@]}; do
        if ! python3 -c "import $pkg" &> /dev/null; then
            if [[ -z ${APT_UPDATED} ]]; then
                say "Running apt update.."
                apt update > /dev/null 2>&1

                APT_UPDATED=true
            fi

            say "Installing python3-${pkg}..."
            apt install -y python3-${pkg} > /dev/null 2>&1
        fi
    done
}


function install_golang {
    ! [[ -z $(which go) ]] && return 0

    PACKAGE=go1.22.5.linux-amd64.tar.gz
    URL="https://go.dev/dl/${PACKAGE}"
    PATHS=(
        PATH=${PATH}:/usr/local/go/bin:${HOME}/go/bin
        GOROOT=/usr/local/go
        GOPATH=${HOME}/go
    )

    say "Installing ${PACKAGE}.."

    wget ${URL} > /dev/null 2>&1
    rm -Rf /usr/local/go
    tar -C /usr/local -xzf ${PACKAGE}

    for path in ${PATHS[@]}; do
        update_bashrc "export ${path}"

        export "${path}"
    done

    source ${HOME}/.bashrc
    rm -Rf ${PACKAGE}

    say "Installation completed!"
}


function install_k0sctl {
    ! [[ -z $(which k0sctl) ]] && return 0

    URL="github.com/k0sproject/k0sctl@latest"

    say "Installing ${URL}.."

    go install ${URL} > /dev/null 2>&1
    ! [[ $? == 0 ]] && fail "Installation failed!"

    k0sctl completion > /etc/bash_completion.d/k0sctl

    say "Installation completed!"
}


function install_k8s_extensions {
    if [[ -z $(which kubectl) ]]; then
        URL="https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

        say "Installing ${URL}.."

        curl -LO "${URL}" > /dev/null 2>&1
        chmod +x ./kubectl
        sudo mv ./kubectl /usr/local/bin/kubectl

        say "Installation completed!"
    fi

    if [[ -z $(which helm) ]]; then
        URL="https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3"

        say "Installing ${URL}.."

        curl "${URL}" | bash

        say "Installation completed!"
    fi
}


function install_fluxcd {
    if [[ -z $(which flux) ]]; then
        URL="https://fluxcd.io/install.sh"

        say "Installing FluxCD.."

        curl -s ${URL} | sudo bash
        . <(flux completion bash)
        ! [[ $? == 0 ]] && fail "Installation failed!"

        say "Installation completed!"
    fi

    [[ -z $1 ]] && fail "FluxCD parameters must be passed as an argument to install_fluxcd()!"

    say "Bootstrapping FluxCD into ${K0SCTL_CLUSTER_NAME}.."

    ARGS="--silent $1"

    flux check --pre --kubeconfig="${K0SCTL_KUBECONFIG_PATH}"
    flux bootstrap git --kubeconfig="${K0SCTL_KUBECONFIG_PATH}" ${ARGS}
    ! [[ $? == 0 ]] && fail "Bootstrapping failed!"

    say "Bootstrapping completed!"
}
