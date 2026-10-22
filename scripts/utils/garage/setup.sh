#!/bin/sh
set -eu

API="${GARAGE_ADMIN_URL:-http://garage.monitoring.svc:3903}"
BUCKETS="${GARAGE_BUCKETS:-loki-chunks loki-ruler tempo}"
KEY_NAME="${GARAGE_KEY_NAME:-monitoring}"

log() { echo "[$(date -Iseconds)] [garage-setup] $*"; }

api() {
  method="$1"
  path="$2"
  shift 2
  curl -sS --fail-with-body -X "$method" -H "Authorization: Bearer ${GARAGE_ADMIN_TOKEN}" -H "Content-Type: application/json" "$@" "${API}${path}"
}

log "Waiting for a healthy cluster at ${API}"
until api GET /v2/GetClusterHealth 2>/dev/null | grep -q '"status": *"healthy"'; do
  sleep 5
done

if api GET "/v2/GetKeyInfo?id=${S3_ACCESS_KEY_ID}" >/dev/null 2>&1; then
  log "Key ${KEY_NAME} already present"
else
  api POST /v2/ImportKey -d "{\"accessKeyId\":\"${S3_ACCESS_KEY_ID}\",\"secretAccessKey\":\"${S3_SECRET_ACCESS_KEY}\",\"name\":\"${KEY_NAME}\"}" >/dev/null
  log "Imported key ${KEY_NAME}"
fi

for bucket in $BUCKETS; do
  if info="$(api GET "/v2/GetBucketInfo?globalAlias=${bucket}" 2>/dev/null)"; then
    log "Bucket ${bucket} already present"
  else
    info="$(api POST /v2/CreateBucket -d "{\"globalAlias\":\"${bucket}\"}")"
    log "Created bucket ${bucket}"
  fi
  id="$(echo "$info" | tr -d '\n' | sed -n 's/^[^{]*{ *"id": *"\([0-9a-f]*\)".*/\1/p')"
  api POST /v2/AllowBucketKey -d "{\"bucketId\":\"${id}\",\"accessKeyId\":\"${S3_ACCESS_KEY_ID}\",\"permissions\":{\"read\":true,\"write\":true,\"owner\":true}}" >/dev/null
  log "Granted ${KEY_NAME} read/write/owner on ${bucket}"
done

log "Done"
