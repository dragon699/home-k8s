#!/bin/bash

set -euo pipefail

APP="$1"

SA=/var/run/secrets/kubernetes.io/serviceaccount
URL="https://kubernetes.default.svc/apis/argoproj.io/v1alpha1/namespaces/argocd/applications/${APP}"
kube() { curl -sS --fail-with-body --cacert "$SA/ca.crt" -H "Authorization: Bearer $(cat "$SA/token")" "$@"; }

if [ "$(curl -sS -o /dev/null -w '%{http_code}' --cacert "$SA/ca.crt" -H "Authorization: Bearer $(cat "$SA/token")" "$URL")" = "404" ]; then
  echo "${APP} has no Argo CD application (deploy: false?); skipping"
  exit 0
fi

# Wait for a running operation (e.g. an auto-sync) to finish first
for i in $(seq 1 60); do
  [ "$(kube "$URL" | jq -r '.operation != null')" = "false" ] && break
  echo "Waiting for the current operation on ${APP} to finish..."
  sleep 5
done

# A sync requested this way doesn't inherit the app's syncOptions (e.g. ServerSideApply)
PATCH=$(kube "$URL" | jq -c '{operation: {
  initiatedBy: {username: "github-actions"},
  sync: {prune: false, syncOptions: (.spec.syncPolicy.syncOptions // [])},
  retry: {limit: 15, backoff: {duration: "10s", factor: 1, maxDuration: "10s"}}
}}')

STARTED=$(date -u +%Y-%m-%dT%H:%M:%SZ)
kube -X PATCH -H "Content-Type: application/merge-patch+json" "$URL" -d "$PATCH" >/dev/null
echo "Sync of ${APP} requested on $(TZ=Europe/Sofia date +'%d/%m at %H:%M:%S')"

for i in $(seq 1 100); do
  read -r STARTED_AT PHASE < <(kube "$URL" | jq -r '[.status.operationState.startedAt // "", .status.operationState.phase // ""] | @tsv')
  if [[ ! "$STARTED_AT" < "$STARTED" ]] && [[ "$PHASE" =~ ^(Succeeded|Failed|Error)$ ]]; then
    FINISHED=$(kube "$URL" | jq -r '.status.operationState.finishedAt')
    kube "$URL" | jq -r '"\(.status.operationState.phase): \(.status.operationState.message)",
      "sync: \(.status.sync.status), health: \(.status.health.status)"'
    echo "Finished on $(TZ=Europe/Sofia date -d "$FINISHED" +'%d/%m at %H:%M:%S')"
    [ "$PHASE" = "Succeeded" ] && exit 0 || exit 1
  fi
  sleep 5
done
echo "Timed out waiting for the sync of ${APP}"
exit 1
