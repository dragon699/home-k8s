#!/usr/bin/env bash
set -uo pipefail

BASE="$1"
HEAD="$2"
APP="$3"
WORK="$(mktemp -d)"
BIN="${WORK}/bin"
mkdir -p "$BIN"
export PATH="${BIN}:${PATH}"

if ! command -v helm >/dev/null; then
  curl -fsSL https://get.helm.sh/helm-v3.18.4-linux-amd64.tar.gz | tar -xz -C "$WORK" linux-amd64/helm
  mv "${WORK}/linux-amd64/helm" "$BIN/"
fi
curl -fsSL -o "${BIN}/yq" https://github.com/mikefarah/yq/releases/download/v4.45.1/yq_linux_amd64
chmod +x "${BIN}/yq"
export DOCKER_CONFIG="${WORK}/docker" HELM_CACHE_HOME="${WORK}/helm-cache" HELM_CONFIG_HOME="${WORK}/helm-config" HELM_DATA_HOME="${WORK}/helm-data"

git worktree add -q --detach "${WORK}/base" "$BASE"
git worktree add -q --detach "${WORK}/head" "$HEAD"
trap 'git worktree remove --force "${WORK}/base"; git worktree remove --force "${WORK}/head"; rm -rf "$WORK"' EXIT

app_json() {
  cat "$1"/argocd/apps/*.yaml | yq -o=json -I=0 ".projects[][] | select(.name == \"${APP}\")" | head -n1
}

OLD_APP="$(app_json "${WORK}/base")"
NEW_APP="$(app_json "${WORK}/head")"
[[ -z "$NEW_APP" ]] && exit 0

if [[ "$(jq -r '.deploy | tostring' <<< "$NEW_APP")" == "false" ]]; then
  echo "Deployed: no (\`deploy: false\` in argocd/apps), merging has no live effect until the app is enabled."
  echo
fi

NS="$(jq -r '.target.namespace // "default"' <<< "$NEW_APP")"
SA_DIR=/var/run/secrets/kubernetes.io/serviceaccount
if [[ -z "${KUBE_VERSION:-}" && -r "${SA_DIR}/token" ]]; then
  KUBE_VERSION="$(curl -sS --cacert "${SA_DIR}/ca.crt" -H "Authorization: Bearer $(cat "${SA_DIR}/token")" https://kubernetes.default.svc/version 2>/dev/null | jq -r '.gitVersion // empty' | sed 's/+.*//')"
fi
KUBE_ARGS=()
[[ -n "${KUBE_VERSION:-}" ]] && KUBE_ARGS=(--kube-version "$KUBE_VERSION")

fetch_chart() {
  local src="$1" dest="$2" url chart rev
  url="$(jq -r '.repoURL' <<< "$src")"
  chart="$(jq -r '.chart' <<< "$src")"
  rev="$(jq -r '.targetRevision' <<< "$src")"
  mkdir -p "$dest"
  if [[ "$url" == oci://* ]]; then
    helm pull "$url" --version "$rev" --untar --untardir "$dest" >/dev/null 2>&1
  else
    helm pull "$chart" --repo "$url" --version "$rev" --untar --untardir "$dest" >/dev/null 2>&1
  fi
  find "$dest" -mindepth 2 -maxdepth 2 -name Chart.yaml -printf '%h/\n' | head -n1
}

render() {
  local src="$1" root="$2" chartdir="$3" rel args=()
  rel="$(jq -r --arg app "$APP" '.helm.releaseName // $app' <<< "$src")"
  while read -r f; do
    [[ -n "$f" ]] && args+=(-f "${root}/${f#\$values/}")
  done < <(jq -r '.helm.valueFiles[]? // empty' <<< "$src")
  helm template "$rel" "$chartdir" --namespace "$NS" --include-crds "${KUBE_ARGS[@]}" "${args[@]}" 2>&1
}

values_changed() {
  local src="$1" f
  while read -r f; do
    [[ -n "$f" ]] || continue
    git diff --quiet "$BASE" "$HEAD" -- "${f#\$values/}" || return 0
  done < <(jq -r '.helm.valueFiles[]? // empty' <<< "$src")
  return 1
}

SCRIPTS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

check_images() {
  local render="$1" img total=0 missing=() unchecked=()
  while read -r img; do
    [[ -z "$img" ]] && continue
    total=$((total + 1))
    bash "${SCRIPTS}/image-exists.sh" "$img"
    case $? in
      0) ;;
      1) missing+=("$img") ;;
      *) unchecked+=("$img") ;;
    esac
  done < <(yq -N '.. | select(tag == "!!map" and has("image")) | .image | select(tag == "!!str")' "$render" | sort -u)
  for img in "${missing[@]}"; do
    echo "CHECK FAILED: image ${img} does not exist in its registry"
  done
  echo "Images the new version pulls: ${total}, missing: ${#missing[@]}${unchecked:+, not checkable (private or rate-limited): ${unchecked[*]}}"
}

ensure_kubeconform() {
  command -v kubeconform >/dev/null && return
  curl -fsSL https://github.com/yannh/kubeconform/releases/download/v0.7.0/kubeconform-linux-amd64.tar.gz | tar -xz -C "$BIN" kubeconform
}

check_custom_resources() {
  local render="$1" dir="$2" crds f doc group version kind served n=0 bad=0 out
  crds="$(yq -o=json -I=0 'select(.kind == "CustomResourceDefinition")' "$render")"
  [[ -z "$crds" ]] && return
  ensure_kubeconform
  mkdir -p "${dir}/schemas" "${dir}/docs"
  jq -r '.spec.group as $g | .spec.names.kind as $k | .spec.versions[] | select(.served) | [$g, $k, .name] | @tsv' <<< "$crds" > "${dir}/served.tsv"
  while IFS=$'\t' read -r group kind version; do
    mkdir -p "${dir}/schemas/${group}"
    jq -s --arg g "$group" --arg k "$kind" --arg v "$version" '
      [.[] | .spec | select(.group == $g and .names.kind == $k) | .versions[] | select(.name == $v) | .schema.openAPIV3Schema][0]
      | walk(if type == "object" and .type == "object" and has("properties") then
          (if (has("additionalProperties") | not) and ((.["x-kubernetes-preserve-unknown-fields"] // false) | not) then . + {additionalProperties: false} else . end)
          | (.properties as $p | if has("required") then .required |= map(select(($p[.] // {}) | has("default") | not)) else . end)
        else . end)
    ' <<< "$crds" > "${dir}/schemas/${group}/${kind,,}_${version}.json"
  done < "${dir}/served.tsv"

  while read -r f; do
    while read -r doc; do
      [[ -z "$doc" ]] && continue
      IFS=$'\t' read -r group version kind < <(jq -r '(.apiVersion | split("/")) as $a | [($a | if length > 1 then .[0] else "" end), $a[-1], .kind] | @tsv' <<< "$doc")
      awk -F'\t' -v g="$group" -v k="$kind" '$1 == g && $2 == k { found = 1 } END { exit !found }' "${dir}/served.tsv" || continue
      n=$((n + 1))
      if ! awk -F'\t' -v g="$group" -v k="$kind" -v v="$version" '$1 == g && $2 == k && $3 == v { found = 1 } END { exit !found }' "${dir}/served.tsv"; then
        served="$(awk -F'\t' -v g="$group" -v k="$kind" '$1 == g && $2 == k { print $3 }' "${dir}/served.tsv" | paste -sd, -)"
        echo "CHECK FAILED: ${f#"${WORK}/head/"} ${kind} $(jq -r '.metadata.name' <<< "$doc") uses ${group}/${version}, which the new CRDs no longer serve (served: ${served})"
        bad=$((bad + 1))
        continue
      fi
      printf '%s' "$doc" > "${dir}/docs/doc.json"
      if ! out="$(kubeconform -ignore-missing-schemas -schema-location "${dir}/schemas/{{ .Group }}/{{ .ResourceKind }}_{{ .ResourceAPIVersion }}.json" "${dir}/docs/doc.json" 2>&1)"; then
        echo "CHECK FAILED: ${f#"${WORK}/head/"} ${kind} $(jq -r '.metadata.name' <<< "$doc") does not match the new ${group}/${version} schema: $(sed -E "s#^.*(failed with '[^']*' - |is invalid: )##" <<< "$out" | head -c 400 | tr '\n' ' ')"
        bad=$((bad + 1))
      fi
    done < <(yq -o=json -I=0 'select(.kind != null and .kind != "CustomResourceDefinition")' "$f" 2>/dev/null)
  done < <({ find "${WORK}/head/manifests" -type f \( -name '*.yaml' -o -name '*.yml' \) | sort; echo "$render"; })
  echo "Custom resources checked against the new CRDs (repo manifests and this chart's render): ${n}, ${bad} failing"
}

removed_keys() {
  local repo="$1" old="$2" new="$3"
  jq -rn --slurpfile r "$repo" --slurpfile o "$old" --slurpfile n "$new" '
    def has_path($d; $p): ($p | length) == 0 or (($d | try getpath($p[:-1]) catch null) | type == "object" and has($p[-1]));
    [ $r[0] // {} | paths(scalars) | .[:(map(type == "number") | index(true) // length)] ]
    | unique[]
    | . as $p
    | [range(length; 0; -1) as $k | $p[:$k] | select(has_path($o[0]; .))][0] // empty
    | select(has_path($n[0]; .) | not)
    | join(".")' | sort -u
}

COUNT="$(jq '.sources | length' <<< "$NEW_APP")"
for ((i = 0; i < COUNT; i++)); do
  NEW_SRC="$(jq -c ".sources[$i]" <<< "$NEW_APP")"
  OLD_SRC="$(jq -c ".sources[$i] // empty" <<< "$OLD_APP")"
  [[ "$(jq -r '.chart // empty' <<< "$NEW_SRC")" == "" || -z "$OLD_SRC" ]] && continue
  OLD_REV="$(jq -r '.targetRevision' <<< "$OLD_SRC")"
  NEW_REV="$(jq -r '.targetRevision' <<< "$NEW_SRC")"
  [[ "$OLD_REV" == "$NEW_REV" ]] && ! values_changed "$NEW_SRC" && continue

  CHART="$(jq -r '.chart' <<< "$NEW_SRC")"
  REL="$(jq -r --arg app "$APP" '.helm.releaseName // $app' <<< "$NEW_SRC")"
  D="${WORK}/src${i}"
  OLD_DIR="$(fetch_chart "$OLD_SRC" "${D}/old")"
  NEW_DIR="$(fetch_chart "$NEW_SRC" "${D}/new")"
  echo "### Chart ${CHART} ${OLD_REV} → ${NEW_REV} (release ${REL}, namespace ${NS})"
  if [[ -z "$OLD_DIR" || -z "$NEW_DIR" ]]; then
    echo "Could not download one of the chart versions."
    echo
    continue
  fi
  echo "appVersion: $(yq '.appVersion' "${OLD_DIR}Chart.yaml") → $(yq '.appVersion' "${NEW_DIR}Chart.yaml")"

  render "$OLD_SRC" "${WORK}/base" "$OLD_DIR" > "${D}/old.yaml"
  if ! render "$NEW_SRC" "${WORK}/head" "$NEW_DIR" > "${D}/new.yaml"; then
    echo "New version FAILS to render with the repo values:"
    echo '```'
    head -c 3000 "${D}/new.yaml"
    echo '```'
    echo
    continue
  fi

  for side in old new; do
    yq 'select(.kind != "CustomResourceDefinition")' "${D}/${side}.yaml" | grep -vE '^\s*(helm\.sh/chart|app\.kubernetes\.io/version):' > "${D}/${side}.main.yaml"
    yq -N 'select(.kind == "CustomResourceDefinition") | .metadata.name' "${D}/${side}.yaml" | sort > "${D}/${side}.crds"
  done

  for side in old new; do
    yq -o=json "${D}/${side}/"*/values.yaml > "${D}/${side}.defaults.json" 2>/dev/null || echo '{}' > "${D}/${side}.defaults.json"
  done
  : > "${D}/repo.json"
  while read -r f; do
    [[ -n "$f" ]] && yq -o=json "${WORK}/head/${f#\$values/}" >> "${D}/repo.json"
  done < <(jq -r '.helm.valueFiles[]? // empty' <<< "$NEW_SRC")
  [[ -s "${D}/repo.json" ]] || echo '{}' > "${D}/repo.json"
  jq -s 'reduce .[] as $x ({}; . * $x)' "${D}/repo.json" > "${D}/repo.merged.json"
  REMOVED="$(removed_keys "${D}/repo.merged.json" "${D}/old.defaults.json" "${D}/new.defaults.json")"
  if [[ -n "$REMOVED" ]]; then
    echo "Repo values keys that no longer exist in the new chart defaults (now ignored):"
    sed 's/^/- /' <<< "$REMOVED"
  else
    echo "Every repo values key still exists in the new chart defaults."
  fi

  ADDED_CRDS="$(comm -13 "${D}/old.crds" "${D}/new.crds" | paste -sd, -)"
  GONE_CRDS="$(comm -23 "${D}/old.crds" "${D}/new.crds" | paste -sd, -)"
  CHANGED_CRDS=""
  while read -r crd; do
    [[ -z "$crd" ]] && continue
    if ! diff -q <(yq "select(.metadata.name == \"${crd}\")" "${D}/old.yaml") <(yq "select(.metadata.name == \"${crd}\")" "${D}/new.yaml") >/dev/null; then
      CHANGED_CRDS+="${crd},"
    fi
  done < <(comm -12 "${D}/old.crds" "${D}/new.crds")
  [[ -n "$ADDED_CRDS$GONE_CRDS$CHANGED_CRDS" ]] && echo "CRDs: added [${ADDED_CRDS}] removed [${GONE_CRDS}] changed [${CHANGED_CRDS%,}]"

  for side in old new; do
    yq -N 'select(.kind == "Deployment" or .kind == "StatefulSet" or .kind == "DaemonSet") | .kind + "/" + .metadata.name + " " + (.spec.template | to_json | @base64)' "${D}/${side}.yaml" | sort > "${D}/${side}.pods"
  done
  RESTARTS="$(comm -13 "${D}/old.pods" "${D}/new.pods" | cut -d' ' -f1 | paste -sd, -)"
  echo "Workloads whose pods restart on sync: ${RESTARTS:-none}"
  check_images "${D}/new.yaml"
  check_custom_resources "${D}/new.yaml" "${D}/crs"
  if [[ -n "${RENDER_DIR:-}" ]]; then
    mkdir -p "${RENDER_DIR}/old" "${RENDER_DIR}/new"
    cp "${D}/old.yaml" "${RENDER_DIR}/old/src${i}.yaml"
    cp "${D}/new.yaml" "${RENDER_DIR}/new/src${i}.yaml"
  fi

  echo
  if diff -q "${D}/old.main.yaml" "${D}/new.main.yaml" >/dev/null; then
    echo "Rendered manifests are identical apart from chart/version labels and CRDs."
  else
    echo "Rendered manifest diff with the repo values (CRDs and chart/version labels excluded):"
    echo '```diff'
    diff -u --label "${OLD_REV}" --label "${NEW_REV}" "${D}/old.main.yaml" "${D}/new.main.yaml" | head -c 12000
    echo '```'
  fi
  echo
done
