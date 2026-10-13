#!/bin/bash


function expand_cron_field() {
    local PART START END STEP

    echo "${1}" | tr ',' '\n' | while read -r PART; do
        STEP=1
        case "${PART}" in
            */*) STEP="${PART#*/}"; PART="${PART%/*}" ;;
        esac

        case "${PART}" in
            '*') START="${2}"; END="${3}" ;;
            *-*) START="${PART%-*}"; END="${PART#*-}" ;;
            *) START="${PART}"; END="${PART}" ;;
        esac

        seq "${START}" "${STEP}" "${END}"
    done | sed 's/^7$/0/' | sort -n | uniq | tr '\n' ' '
}

function cron_next_run() {
    local MINUTES HOURS DAYS MONTHS WEEKDAYS ANY_DAY ANY_WEEKDAY
    local NOW_HOUR NOW_MINUTE OFFSET DATE DAY MONTH WEEKDAY DAY_MATCH WEEKDAY_MATCH HOUR MINUTE

    set -f
    set -- ${1}
    set +f

    MINUTES="$(expand_cron_field "${1}" 0 59)"
    HOURS="$(expand_cron_field "${2}" 0 23)"
    DAYS="$(expand_cron_field "${3}" 1 31)"
    MONTHS="$(expand_cron_field "${4}" 1 12)"
    WEEKDAYS="$(expand_cron_field "${5}" 0 7)"
    [ "${3}" = '*' ] && ANY_DAY=1 || ANY_DAY=0
    [ "${5}" = '*' ] && ANY_WEEKDAY=1 || ANY_WEEKDAY=0

    NOW_HOUR="$(date +%-H)"
    NOW_MINUTE="$(date +%-M)"

    for OFFSET in $(seq 0 366); do
        set -- $(date -d "today +${OFFSET} day" +'%F %-d %-m %w')
        DATE="${1}"; DAY="${2}"; MONTH="${3}"; WEEKDAY="${4}"

        case " ${MONTHS}" in *" ${MONTH} "*) ;; *) continue ;; esac

        case " ${DAYS}" in *" ${DAY} "*) DAY_MATCH=1 ;; *) DAY_MATCH=0 ;; esac
        case " ${WEEKDAYS}" in *" ${WEEKDAY} "*) WEEKDAY_MATCH=1 ;; *) WEEKDAY_MATCH=0 ;; esac

        if [ "${ANY_DAY}" -eq 1 ] || [ "${ANY_WEEKDAY}" -eq 1 ]; then
            [ "${DAY_MATCH}" -eq 1 ] && [ "${WEEKDAY_MATCH}" -eq 1 ] || continue
        else
            [ "${DAY_MATCH}" -eq 1 ] || [ "${WEEKDAY_MATCH}" -eq 1 ] || continue
        fi

        for HOUR in ${HOURS}; do
            [ "${OFFSET}" -eq 0 ] && [ "${HOUR}" -lt "${NOW_HOUR}" ] && continue

            for MINUTE in ${MINUTES}; do
                [ "${OFFSET}" -eq 0 ] && [ "${HOUR}" -eq "${NOW_HOUR}" ] && [ "${MINUTE}" -le "${NOW_MINUTE}" ] && continue

                date -d "${DATE} ${HOUR}:${MINUTE}" +%s
                return 0
            done
        done
    done

    return 1
}

function cronjob_metrics_start() {
    JOB_NAME="${1}"
    JOB_SCHEDULE="$(grep -vE '^[[:space:]]*(#|$)|^[A-Za-z_]+=' "${2}" | awk '{print $1, $2, $3, $4, $5; exit}')"
    JOB_METRICS_FILE="${3}"
    JOB_LAST_SUCCESS="$(awk '/^host_cronjob_status_last_successful_time/ {print $2}' "${JOB_METRICS_FILE}" 2>/dev/null)"
    JOB_LAST_SCHEDULE="$(date +%s)"

    write_cronjob_metrics 1
    trap 'cronjob_metrics_finish ${?}' EXIT
}

function cronjob_metrics_finish() {
    [ "${1}" -eq 0 ] && JOB_LAST_SUCCESS="$(date +%s)"

    write_cronjob_metrics 0
}

function write_cronjob_metrics() {
    local NEXT_SCHEDULE="$(cron_next_run "${JOB_SCHEDULE}")"
    local LABELS="cronjob=\"${JOB_NAME}\""

    mkdir -p "$(dirname "${JOB_METRICS_FILE}")"

    {
        echo "# TYPE host_cronjob_info gauge"
        echo "host_cronjob_info{${LABELS},schedule=\"${JOB_SCHEDULE}\",timezone=\"local\"} 1"
        echo "# TYPE host_cronjob_status_active gauge"
        echo "host_cronjob_status_active{${LABELS}} ${1}"
        echo "# TYPE host_cronjob_status_last_schedule_time gauge"
        echo "host_cronjob_status_last_schedule_time{${LABELS}} ${JOB_LAST_SCHEDULE}"

        if [ -n "${JOB_LAST_SUCCESS}" ]; then
            echo "# TYPE host_cronjob_status_last_successful_time gauge"
            echo "host_cronjob_status_last_successful_time{${LABELS}} ${JOB_LAST_SUCCESS}"
        fi

        if [ -n "${NEXT_SCHEDULE}" ]; then
            echo "# TYPE host_cronjob_next_schedule_time gauge"
            echo "host_cronjob_next_schedule_time{${LABELS}} ${NEXT_SCHEDULE}"
        fi
    } > "${JOB_METRICS_FILE}.tmp"

    mv "${JOB_METRICS_FILE}.tmp" "${JOB_METRICS_FILE}"
}
