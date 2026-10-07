#!/bin/bash


function main() {
    echo "Installing cron job for k0s backups.."

    sudo install -o root -g root -m 644 "$(dirname "${0}")/cron" /etc/cron.d/backup-k0s
    [[ ${?} -ne 0 ]] && echo "Failed to install cron job!" && exit 1

    echo "Cron job installed successfully!"

}

main
