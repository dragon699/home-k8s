#!/bin/bash

# Usage: ./decrypt-file.sh <file.age>


AGE_KEY_FILE_PATH="${HOME}/age-backups-key.txt.age"

INPUT_FILE="${1}"
OUTPUT_FILE="${INPUT_FILE%.age}"


[[ -z "${INPUT_FILE}" ]] && echo "Usage: decrypt-file.sh <file.age>" && exit 1

[[ "${INPUT_FILE}" != *.age ]] && echo "${INPUT_FILE} is not an .age file!" && exit 1

age -d -i "${AGE_KEY_FILE_PATH}" -o "${OUTPUT_FILE}" "${INPUT_FILE}"
