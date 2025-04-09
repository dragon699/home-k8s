#!/bin/bash

# !!
# This script must be run as root user;
# !!

# Script containing functions to ensure prerequisites for some system settings;
# that fixes known issues for Longhorn and Grafana Alloy/Loki;


function ensure_longhorn_prerequisites() {
    # This fixes multi-mount issues with PV volumes;
    # Reference -> https://longhorn.io/kb/troubleshooting-volume-with-multipath/
    function fix_multipath() {
        echo "Verifying multipathd settings.."

        MULTIPATH_CONF='/etc/multipath.conf'
        MULTIPATH_STR_RE_CHECK='blacklist {\n( )+devnode( )+\"\^sd\[a\-z0\-9\]\+\"'
        MULTIPATH_STR=$(cat <<EOF
blacklist {
    devnode "^sd[a-z0-9]+"
}
EOF
        )

        if ! systemctl list-unit-files --type=service | grep -q multipathd.service; then
            return 0
        fi

        ! [[ -f ${MULTIPATH_CONF} ]] && touch ${MULTIPATH_CONF}
        
        if ! grep -Pzoq "${MULTIPATH_STR_RE_CHECK}" ${MULTIPATH_CONF}; then
            echo -e "${MULTIPATH_STR}" >> ${MULTIPATH_CONF}

            systemctl restart multipathd.service
        fi
    }

    function verify_packages() {
        EXISTING_PACKAGES=$(dpkg -l)
        PACKAGES=(
            open-iscsi
            nfs-common
        )

        for pkg in ${PACKAGES[@]}; do
            if ! echo ${EXISTING_PACKAGES} | grep -q ${pkg}; then
                if [[ -z ${APT_UPDATED} ]]; then
                    apt update

                    APT_UPDATED=true
                fi

                apt install -y ${pkg}
            fi
        done

        systemctl enable iscsid
        systemctl start iscsid
    }

    fix_multipath
    verify_packages
}


function ensure_alloy_prequisites() {
    # This fixes a spam error that occurs in logs;
    # known as `failed to create fsnotify watcher: too many open files`;
    # caused by the default inotify limits;
    function fix_inotify_limits() {
        echo "Verifying inotify settings.."

        INOTIFY_CONF='/etc/sysctl.d/99-inotify.conf'

        MAX_USER_WATCHES=2099999999
        MAX_USER_INSTANCES=2099999999
        MAX_QUEUED_EVENTS=2099999999

        printf "fs.inotify.max_user_watches=${MAX_USER_WATCHES}\nfs.inotify.max_user_instances=${MAX_USER_INSTANCES}\nfs.inotify.max_queued_events=${MAX_QUEUED_EVENTS}" > ${INOTIFY_CONF}

        sysctl -p --system > /dev/null 2>&1
    }

    fix_inotify_limits
}


ensure_longhorn_prerequisites
ensure_alloy_prequisites
