#!/bin/bash

# Usage: ./show-public-key.sh


AGE_KEY_FILE_PATH="./age-backups-key.txt.age"

age -d ${AGE_KEY_FILE_PATH} | age-keygen -y
