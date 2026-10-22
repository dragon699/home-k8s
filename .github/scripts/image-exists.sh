#!/usr/bin/env bash
set -uo pipefail

REF="$1"
NAME="${REF%@*}"
if [[ "$REF" == *@* ]]; then
  REFERENCE="${REF#*@}"
  [[ "${NAME##*/}" == *:* ]] && NAME="${NAME%:*}"
elif [[ "${NAME##*/}" == *:* ]]; then
  REFERENCE="${NAME##*:}"
  NAME="${NAME%:*}"
else
  REFERENCE=latest
fi

FIRST="${NAME%%/*}"
if [[ "$NAME" == */* && ( "$FIRST" == *.* || "$FIRST" == *:* || "$FIRST" == localhost ) ]]; then
  REGISTRY="$FIRST"
  REPOSITORY="${NAME#*/}"
else
  REGISTRY=docker.io
  REPOSITORY="$NAME"
fi
if [[ "$REGISTRY" == docker.io ]]; then
  REGISTRY=registry-1.docker.io
  [[ "$REPOSITORY" != */* ]] && REPOSITORY="library/${REPOSITORY}"
fi

URL="https://${REGISTRY}/v2/${REPOSITORY}/manifests/${REFERENCE}"
ACCEPT="application/vnd.oci.image.index.v1+json,application/vnd.docker.distribution.manifest.list.v2+json,application/vnd.docker.distribution.manifest.v2+json,application/vnd.oci.image.manifest.v1+json"

head_status() {
  curl -sS -L -o /dev/null -I -m 30 -w '%{http_code}' -H "Accept: ${ACCEPT}" "$@" "$URL" 2>/dev/null
}

STATUS="$(head_status)"
if [[ "$STATUS" == 401 ]]; then
  CHALLENGE="$(curl -sS -I -m 30 -H "Accept: ${ACCEPT}" "$URL" 2>/dev/null | tr -d '\r' | grep -i '^www-authenticate:' | head -n1)"
  REALM="$(sed -nE 's/.*realm="([^"]+)".*/\1/p' <<< "$CHALLENGE")"
  SERVICE="$(sed -nE 's/.*service="([^"]+)".*/\1/p' <<< "$CHALLENGE")"
  SCOPE="$(sed -nE 's/.*scope="([^"]+)".*/\1/p' <<< "$CHALLENGE")"
  [[ -z "$SCOPE" ]] && SCOPE="repository:${REPOSITORY}:pull"
  if [[ -n "$REALM" ]]; then
    TOKEN="$(curl -sS -m 30 -G "$REALM" --data-urlencode "service=${SERVICE}" --data-urlencode "scope=${SCOPE}" 2>/dev/null | jq -r '.token // .access_token // empty' 2>/dev/null)"
    [[ -n "$TOKEN" ]] && STATUS="$(head_status -H "Authorization: Bearer ${TOKEN}")"
  fi
fi

case "$STATUS" in
  200) exit 0 ;;
  404) exit 1 ;;
  *) exit 2 ;;
esac
