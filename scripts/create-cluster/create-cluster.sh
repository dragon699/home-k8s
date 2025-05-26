#!/bin/bash

# !!
# This script must be run as root user;
# `sudo` will not work;
# !!

# - Edit parameters in config.yaml located in the same directory;
# - Run: ./create-cluster.sh to provision and deploy the entire kubernetes cluster;
#   If parameters under `FluxCD configuration` were set;
#   Flux (CD) will also be installed into the cluster;


SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
LIB_DIR="${SCRIPT_DIR}/../lib"

source "${LIB_DIR}/installs.sh"
ensure_root


function set_env {
    install_bash_utils

    export SSH_KNOWN_HOSTS="/dev/null"

    export K0SCTL_CONFIG_TEMPLATE_PATH="${LIB_DIR}/templates/create-cluster/k0s-cluster.yaml.j2"
    export K0SCTL_CONFIG_INPUTS_PATH="${SCRIPT_DIR}/config.yaml"
    export K0SCTL_CLUSTER_NAME="$(yq e .cluster_name "${K0SCTL_CONFIG_INPUTS_PATH}")"
    export K0SCTL_CLUSTER_IS_DEFAULT="$(yq e .cluster_is_default "${K0SCTL_CONFIG_INPUTS_PATH}")"

    [[ -z "${K0SCTL_CLUSTER_NAME}" ]] && \
    fail "${K0SCTL_CONFIG_INPUTS_PATH}: Invalid configuration!"

    export K0SCTL_CLUSTER_DIR="${HOME}/${K0SCTL_CLUSTER_NAME}"
    export K0SCTL_DESTROY_SCRIPT_TEMPLATE_PATH="${LIB_DIR}/templates/create-cluster/destroy-cluster.sh.j2"
    export K0SCTL_CONFIG_PATH="${K0SCTL_CLUSTER_DIR}/k0sctl.yaml"
    export K0SCTL_DESTROY_SCRIPT_PATH="${K0SCTL_CLUSTER_DIR}/destroy-cluster.sh"
    export K0SCTL_KUBECONFIG_PATH="${K0SCTL_CLUSTER_DIR}/kubeconfig"

    export FLUXCD_ENABLED="$(yq e .install_fluxcd "${K0SCTL_CONFIG_INPUTS_PATH}")"
    export FLUXCD_IMAGE_AUTOMATION="$(yq e .install_fluxcd_image_automation "${K0SCTL_CONFIG_INPUTS_PATH}")"

    export REMOTE_SCRIPT="${LIB_DIR}/kubernetes-prerequisites.sh"
}


function create_cluster {
    install_python_dependencies
    install_golang
    install_k0sctl
    install_k8s_extensions
    set_env

    say "Provisioning ${K0SCTL_CLUSTER_NAME} into ${K0SCTL_CLUSTER_DIR}.."
    python3 "${LIB_DIR}/utils.py" --create-k0sctl-config

    k0sctl apply --config ${K0SCTL_CONFIG_PATH}
    ! [[ $? == 0 ]] && fail "Provisioning failed!"

    python3 "${LIB_DIR}/utils.py" --create-destroy-script
    say "${K0SCTL_DESTROY_SCRIPT_PATH}: destroy script created."

    mkdir -p "${HOME}/.kube"
    k0sctl kubeconfig --config "${K0SCTL_CONFIG_PATH}" > "${K0SCTL_KUBECONFIG_PATH}"
    say "${K0SCTL_KUBECONFIG_PATH}: kubectl config file created."

    if [[ ${K0SCTL_CLUSTER_IS_DEFAULT} == true ]]; then
        cp "${K0SCTL_KUBECONFIG_PATH}" "${HOME}/.kube/config"
        say "${HOME}/.kube/config: kubectl config file set as default."
    fi

    say "Running ${REMOTE_SCRIPT}.."
    python3 "${LIB_DIR}/utils.py" --apply-vm-kubernetes-prerequisites
    ! [[ $? == 0 ]] && fail "Kubernetes post-requirements setup failed!"

    if [[ ${FLUXCD_ENABLED} == true ]]; then
        FLUXCD_ARGS="$(python3 "${LIB_DIR}/utils.py" --get-flux-params)"

        if [[ ${FLUXCD_IMAGE_AUTOMATION} == true ]]; then
            FLUXCD_ARGS="${FLUXCD_ARGS} --components-extra=image-reflector-controller,image-automation-controller"
        fi

        install_fluxcd "${FLUXCD_ARGS}"
    fi

    say "Cluster created successfully!" nl

    if [[ ${K0SCTL_CLUSTER_IS_DEFAULT} == true ]]; then
        say "Now try interacting with ${K0SCTL_CLUSTER_NAME}:"

    else
        say "To interact with ${K0SCTL_CLUSTER_NAME}, use your kubectl config path:"
        say_cmd "export KUBECONFIG=${K0SCTL_KUBECONFIG_PATH}"
        say "Then:"

    fi

    say_cmd "kubectl get nodes"
}


create_cluster
