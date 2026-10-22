#!/usr/bin/env bash
set -uo pipefail

BASE="$1"
HEAD="$2"
APP="$3"
RENDER_DIR="${4:-}"
SCRIPTS="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BOOT_SECONDS="${BOOT_SECONDS:-25}"
read -ra DOCKER_CMD <<< "${DOCKER:-docker}"
WORK="$(mktemp -d -p "${RUNNER_TEMP:-/tmp}")"
trap '"${DOCKER_CMD[@]}" rm -f $("${DOCKER_CMD[@]}" ps -aq --filter "label=app-checks=$$") >/dev/null 2>&1; rm -rf "$WORK"' EXIT
mkdir -p "${WORK}/bin"
export PATH="${WORK}/bin:${PATH}" DOCKER_CONFIG="${WORK}/docker"
curl -fsSL -o "${WORK}/bin/yq" https://github.com/mikefarah/yq/releases/download/v4.45.1/yq_linux_amd64
chmod +x "${WORK}/bin/yq"

echo "### Automated checks"
if ! "${DOCKER_CMD[@]}" version >/dev/null 2>&1; then
  echo "- Skipped: no container runtime."
  exit 0
fi

while read -r img; do
  [[ -z "$img" ]] && continue
  bash "${SCRIPTS}/image-exists.sh" "$img"
  case $? in
    0) echo "- Image ${img}: exists" ;;
    1) echo "CHECK FAILED: image ${img} does not exist in its registry" ;;
    *) echo "- Image ${img}: not checkable (auth or rate limit)" ;;
  esac
done < <(git diff "$BASE" "$HEAD" -- manifests | grep -E '^\+\s+(-\s+)?image:' | sed -E 's/^\+\s+(-\s+)?image:\s*//; s/["'"'"']//g' | sort -u)

rev_of() { if [[ "$1" == old ]]; then echo "$BASE"; else echo "$HEAD"; fi; }

collect_docs() {
  local side="$1" rev f
  rev="$(rev_of "$side")"
  if [[ -n "$RENDER_DIR" ]] && compgen -G "${RENDER_DIR}/${side}/*.yaml" >/dev/null; then
    yq -o=json -I=0 'select(.kind != null)' "${RENDER_DIR}/${side}"/*.yaml 2>/dev/null
  fi
  git ls-tree -r --name-only "$rev" -- manifests | grep -E "^manifests/[^/]+/${APP}/.+\.ya?ml$" | while read -r f; do
    git show "${rev}:${f}" | yq -o=json -I=0 'select(.kind != null)' 2>/dev/null
  done
}

collect_configmaps() {
  local side="$1" rev k dir gen data entry key path line tmp="${WORK}/content"
  rev="$(rev_of "$side")"
  jq -c 'select(.kind == "ConfigMap") | {name: .metadata.name, source: "rendered", data: (.data // {})}' "${WORK}/${side}.docs"
  git ls-tree -r --name-only "$rev" -- config | grep -E '(^|/)kustomization\.ya?ml$' | while read -r k; do
    dir="$(dirname "$k")"
    git show "${rev}:${k}" | yq -o=json -I=0 '.configMapGenerator[]? | {"name": .name, "files": (.files // []), "envs": (.envs // []), "literals": (.literals // [])}' 2>/dev/null | while read -r gen; do
      data='{}'
      while read -r entry; do
        [[ -z "$entry" ]] && continue
        if [[ "$entry" == *=* ]]; then key="${entry%%=*}"; path="${entry#*=}"; else key="$(basename "$entry")"; path="$entry"; fi
        git show "${rev}:${dir}/${path}" > "$tmp" 2>/dev/null || continue
        data="$(jq -c --arg k "$key" --rawfile v "$tmp" '. + {($k): $v}' <<< "$data")"
      done < <(jq -r '.files[]' <<< "$gen")
      while read -r entry; do
        [[ -z "$entry" ]] && continue
        while IFS= read -r line; do
          [[ "$line" =~ ^[[:space:]]*(#|$) ]] && continue
          data="$(jq -c --arg k "${line%%=*}" --arg v "${line#*=}" '. + {($k): $v}' <<< "$data")"
        done < <(git show "${rev}:${dir}/${entry}" 2>/dev/null)
      done < <(jq -r '.envs[]' <<< "$gen")
      while read -r entry; do
        [[ -n "$entry" ]] && data="$(jq -c --arg k "${entry%%=*}" --arg v "${entry#*=}" '. + {($k): $v}' <<< "$data")"
      done < <(jq -r '.literals[]' <<< "$gen")
      jq -c -n --arg n "$(jq -r '.name' <<< "$gen")" --arg s "${dir}/kustomization.yaml" --argjson d "$data" '{name: $n, source: $s, data: $d}'
    done
  done
}

collect_plans() {
  local side="$1"
  jq -c --slurpfile cms "${WORK}/${side}.cms" '
    ([$cms[] | select((.data | length) > 0) | .name]) as $known
    | select(.kind | test("^(Deployment|StatefulSet|DaemonSet|Job|CronJob)$"))
    | (if .kind == "CronJob" then .spec.jobTemplate.spec.template.spec else .spec.template.spec end) as $s
    | .kind as $kind | .metadata.name as $workload
    | $s.containers[]?
    | {kind: $kind, workload: $workload, container: .name, image, command: (.command // []), args: (.args // []),
       env: [.env[]? | select(.value != null) | {name, value: (.value | tostring)}],
       envUnknown: [.env[]? | select(.value == null) | .name],
       envFrom: [.envFrom[]? | .configMapRef.name // empty | select(IN($known[]))],
       mounts: [.volumeMounts[]? as $m | ($s.volumes[]? | select(.name == $m.name)) as $v
         | ([$v.configMap // empty] + [$v.projected.sources[]?.configMap // empty] | map(select(.name | IN($known[])))) as $refs
         | select($refs | length > 0)
         | {mountPath: $m.mountPath, subPath: ($m.subPath // ""), refs: [$refs[] | {name, items: (.items // [])}]}]}
    | select((.mounts | length) > 0 or (.envFrom | length) > 0)
  ' "${WORK}/${side}.docs"
}

materialize() {
  local side="$1" plan="$2" dest="$3" i=0 m ref name items key path dir data
  RUN_ARGS=()
  FILES=()
  mkdir -p "$dest"
  while read -r m; do
    [[ -z "$m" ]] && continue
    i=$((i + 1))
    dir="${dest}/m${i}"
    mkdir -p "$dir"
    while read -r ref; do
      name="$(jq -r '.name' <<< "$ref")"
      data="$(jq -c --arg n "$name" 'select(.name == $n) | .data' "${WORK}/${side}.cms" | head -n1)"
      items="$(jq -c '.items' <<< "$ref")"
      while IFS=$'\t' read -r key path; do
        [[ -z "$key" ]] && continue
        mkdir -p "$(dirname "${dir}/${path}")"
        jq -j --arg k "$key" '.[$k]' <<< "$data" > "${dir}/${path}"
        FILES+=("${name}"$'\t'"${key}"$'\t'"${dir}/${path}"$'\t'"$(jq -r --arg p "$path" 'if .subPath == "" then .mountPath + "/" + $p elif .subPath == $p then .mountPath else "" end' <<< "$m")")
      done < <(if [[ "$items" == "[]" ]]; then jq -r 'keys[] | [., .] | @tsv' <<< "$data"; else jq -r '.[] | [.key, .path] | @tsv' <<< "$items"; fi)
    done < <(jq -c '.refs[]' <<< "$m")
    if [[ "$(jq -r '.subPath' <<< "$m")" == "" ]]; then
      RUN_ARGS+=(-v "${dir}:$(jq -r '.mountPath' <<< "$m"):ro")
    elif [[ -e "${dir}/$(jq -r '.subPath' <<< "$m")" ]]; then
      RUN_ARGS+=(-v "${dir}/$(jq -r '.subPath' <<< "$m"):$(jq -r '.mountPath' <<< "$m"):ro")
    fi
  done < <(jq -c '.mounts[]' <<< "$plan")
  while read -r name; do
    [[ -z "$name" ]] && continue
    while IFS=$'\t' read -r key path; do
      [[ -n "$key" ]] && RUN_ARGS+=(-e "${key}=${path}")
    done < <(jq -r --arg n "$name" 'select(.name == $n) | .data | to_entries[] | [.key, .value] | @tsv' "${WORK}/${side}.cms")
  done < <(jq -r '.envFrom[]' <<< "$plan")
  while IFS=$'\t' read -r key path; do
    [[ -n "$key" ]] && RUN_ARGS+=(-e "${key}=${path}")
  done < <(jq -r '.env[] | [.name, .value] | @tsv' <<< "$plan")
  local known=" ${RUN_ARGS[*]} " var
  while read -r var; do
    [[ -z "$var" || "$known" == *" ${var}="* ]] && continue
    RUN_ARGS+=(-e "${var}=placeholder")
    known+=" ${var}= "
  done < <({ jq -r '.envUnknown[]' <<< "$plan"; cat "${dest}"/m*/* 2>/dev/null | grep -ohE '\$\{[A-Za-z_][A-Za-z0-9_]*' | cut -c3-; } | sort -u)
  chmod -R a+rX "$dest"
}

container_run() {
  local name="$1" mode="$2" out rc
  shift 2
  if [[ "$mode" == check ]]; then
    out="$(timeout 300 "${DOCKER_CMD[@]}" run --rm --quiet --network none --memory 2g --label "app-checks=$$" "$@" 2>&1)"
    rc=$?
    printf '%s\t%s' "$rc" "$(tail -n 6 <<< "$out" | tr '\n\t' '  ' | head -c 600)"
    return
  fi
  "${DOCKER_CMD[@]}" run -d --quiet --name "$name" --network none --memory 2g --label "app-checks=$$" "$@" >/dev/null 2>&1 || { printf 'error\tcould not start the container'; return; }
  for ((t = 0; t < BOOT_SECONDS; t += 2)); do
    [[ "$("${DOCKER_CMD[@]}" inspect -f '{{.State.Running}}' "$name" 2>/dev/null)" == "true" ]] || break
    sleep 2
  done
  out="$("${DOCKER_CMD[@]}" logs --tail 6 "$name" 2>&1 | tr '\n\t' '  ' | head -c 600)"
  if [[ "$("${DOCKER_CMD[@]}" inspect -f '{{.State.Running}}' "$name" 2>/dev/null)" == "true" ]]; then
    printf 'running\t%s' "$out"
  else
    printf 'exited %s\t%s' "$("${DOCKER_CMD[@]}" inspect -f '{{.State.ExitCode}}' "$name" 2>/dev/null)" "$out"
  fi
  "${DOCKER_CMD[@]}" rm -f "$name" >/dev/null 2>&1
}

entrypoint_args() {
  local plan="$1"
  ENTRY=()
  if [[ "$(jq '.command | length' <<< "$plan")" -gt 0 ]]; then
    ENTRY=(--entrypoint "$(jq -r '.command[0]' <<< "$plan")")
  fi
  mapfile -t CMD_ARGS < <(jq -r '.command[1:][], .args[]' <<< "$plan")
}

for side in old new; do
  collect_docs "$side" > "${WORK}/${side}.docs"
  collect_configmaps "$side" > "${WORK}/${side}.cms"
  collect_plans "$side" > "${WORK}/${side}.plans"
done

if [[ ! -s "${WORK}/new.plans" ]]; then
  echo "- No container of ${APP} mounts a ConfigMap from this repo; nothing to validate."
  exit 0
fi

old_file_of() {
  local cm="$1" key="$2" out="$3"
  jq -e -j --arg n "$cm" --arg k "$key" 'select(.name == $n) | .data | select(has($k)) | .[$k]' "${WORK}/old.cms" > "$out" 2>/dev/null
}

prepare_old() {
  [[ -n "$OLD_READY" || -z "$OLD_PLAN" ]] && return
  materialize old "$OLD_PLAN" "${WORK}/plans/old${n}"
  OLD_ARGS=("${RUN_ARGS[@]}")
  OLD_FILES=("${FILES[@]}")
  entrypoint_args "$OLD_PLAN"
  OLD_ENTRY=("${ENTRY[@]}")
  OLD_CMD_ARGS=("${CMD_ARGS[@]}")
  OLD_IMAGE="$(jq -r '.image' <<< "$OLD_PLAN")"
  OLD_READY=1
}

report() {
  local what="$1" detail="$2" old_rc="$3"
  if [[ "$old_rc" == none || "$old_rc" == 0 ]]; then
    echo "CHECK FAILED: ${what}: ${detail}"
  else
    echo "- ${what}: fails on the current version too, so it depends on the live environment, not on this upgrade: ${detail}"
  fi
}

declare -A SEEN_FILE SEEN_CHECK
n=0
while read -r plan; do
  [[ -z "$plan" ]] && continue
  n=$((n + 1))
  KIND="$(jq -r '.kind' <<< "$plan")"
  WORKLOAD="$(jq -r '.workload' <<< "$plan")"
  CONTAINER="$(jq -r '.container' <<< "$plan")"
  IMAGE="$(jq -r '.image' <<< "$plan")"
  LABEL="${KIND}/${WORKLOAD} container ${CONTAINER}"
  OLD_PLAN="$(jq -c --arg k "$KIND" --arg w "$WORKLOAD" --arg c "$CONTAINER" 'select(.kind == $k and .workload == $w and .container == $c)' "${WORK}/old.plans" | head -n1)"
  OLD_READY=""
  materialize new "$plan" "${WORK}/plans/new${n}"
  NEW_ARGS=("${RUN_ARGS[@]}")
  NEW_FILES=("${FILES[@]}")
  entrypoint_args "$plan"
  NEW_ENTRY=("${ENTRY[@]}")
  NEW_CMD_ARGS=("${CMD_ARGS[@]}")

  for entry in "${NEW_FILES[@]}"; do
    IFS=$'\t' read -r cm key host inner <<< "$entry"
    [[ -n "${SEEN_FILE[${cm}/${key}]:-}" ]] && continue
    SEEN_FILE[${cm}/${key}]=1
    case "$key" in
      *.json) parse=(jq empty) ; format=JSON ;;
      *.yaml|*.yml) parse=(yq -N '.') ; format=YAML ;;
      *.toml) parse=(yq -p toml '.') ; format=TOML ;;
      *) continue ;;
    esac
    err="$("${parse[@]}" "$host" 2>&1 >/dev/null)" && continue
    if old_file_of "$cm" "$key" "${WORK}/old-file" && "${parse[@]}" "${WORK}/old-file" >/dev/null 2>&1; then
      echo "CHECK FAILED: ConfigMap ${cm} key ${key} was valid ${format} and no longer is: $(head -c 300 <<< "${err##*: }")"
    fi
  done

  while IFS=$'\037' read -r pattern keypattern vargs ventry; do
    [[ -z "$pattern" || "$pattern" == \#* ]] && continue
    [[ "$IMAGE" =~ $pattern ]] || continue
    read -ra VARGS <<< "$vargs"
    if [[ -z "$keypattern" ]]; then
      id="${IMAGE}|${vargs}|$(jq -c '.args' <<< "$plan")"
      [[ -n "${SEEN_CHECK[$id]:-}" ]] && continue
      SEEN_CHECK[$id]=1
      IFS=$'\t' read -r rc out < <(container_run "" check "${NEW_ARGS[@]}" "${NEW_ENTRY[@]}" "$IMAGE" "${NEW_CMD_ARGS[@]}" "${VARGS[@]}")
      if [[ "$rc" == 0 ]]; then
        echo "- ${LABEL}: \`${vargs}\` passed on ${IMAGE}"
        continue
      fi
      old_rc=none
      prepare_old
      if [[ -n "$OLD_READY" ]]; then
        IFS=$'\t' read -r old_rc _ < <(container_run "" check "${OLD_ARGS[@]}" "${OLD_ENTRY[@]}" "$OLD_IMAGE" "${OLD_CMD_ARGS[@]}" "${VARGS[@]}")
      fi
      report "${LABEL}: \`${vargs}\` on ${IMAGE}" "exit ${rc}: ${out}" "$old_rc"
      continue
    fi
    for entry in "${NEW_FILES[@]}"; do
      IFS=$'\t' read -r cm key host inner <<< "$entry"
      [[ -n "$inner" && "$key" =~ $keypattern ]] || continue
      id="${IMAGE}|${vargs}|${cm}/${key}"
      [[ -n "${SEEN_CHECK[$id]:-}" ]] && continue
      SEEN_CHECK[$id]=1
      OVERRIDE=()
      [[ -n "$ventry" ]] && OVERRIDE=(--entrypoint "$ventry")
      SHOWN="$(sed 's/^ //' <<< "${ventry} ${vargs//\{file\}/$key}")"
      IFS=$'\t' read -r rc out < <(container_run "" check "${NEW_ARGS[@]}" "${OVERRIDE[@]}" "$IMAGE" "${VARGS[@]//\{file\}/$inner}")
      if [[ "$rc" == 0 ]]; then
        echo "- ConfigMap ${cm} key ${key}: \`${SHOWN}\` passed on ${IMAGE}"
        continue
      fi
      old_rc=none
      prepare_old
      if [[ -n "$OLD_READY" ]]; then
        for old_entry in "${OLD_FILES[@]}"; do
          IFS=$'\t' read -r ocm okey ohost oinner <<< "$old_entry"
          [[ "$ocm" == "$cm" && "$okey" == "$key" && -n "$oinner" ]] || continue
          IFS=$'\t' read -r old_rc _ < <(container_run "" check "${OLD_ARGS[@]}" "${OVERRIDE[@]}" "$OLD_IMAGE" "${VARGS[@]//\{file\}/$oinner}")
          break
        done
      fi
      report "ConfigMap ${cm} key ${key}: \`${SHOWN}\` on ${IMAGE}" "exit ${rc}: ${out}" "$old_rc"
    done
  done < <(tr '\t' '\037' < "${SCRIPTS}/config-validators.tsv")

  [[ "$KIND" =~ ^(Deployment|StatefulSet|DaemonSet)$ ]] || continue
  [[ -n "$OLD_PLAN" && "$(jq -r '.image' <<< "$OLD_PLAN")" == "$IMAGE" ]] && continue
  IFS=$'\t' read -r NEW_STATE NEW_OUT < <(container_run "app-checks-$$-new${n}" boot "${NEW_ARGS[@]}" "${NEW_ENTRY[@]}" "$IMAGE" "${NEW_CMD_ARGS[@]}")
  if [[ "$NEW_STATE" == running ]]; then
    echo "- ${LABEL}: ${IMAGE} starts with the repo config and keeps running (${BOOT_SECONDS}s, no network)"
    continue
  fi
  OLD_STATE="no previous version"
  prepare_old
  if [[ -n "$OLD_READY" ]]; then
    IFS=$'\t' read -r OLD_STATE OLD_OUT < <(container_run "app-checks-$$-old${n}" boot "${OLD_ARGS[@]}" "${OLD_ENTRY[@]}" "$OLD_IMAGE" "${OLD_CMD_ARGS[@]}")
  fi
  if [[ "$OLD_STATE" == running ]]; then
    echo "CHECK FAILED: ${LABEL}: ${IMAGE} ${NEW_STATE} on startup with the repo config while the current ${OLD_IMAGE} keeps running under the same conditions: ${NEW_OUT}"
  else
    echo "- ${LABEL}: ${IMAGE} ${NEW_STATE} on startup without network or secrets (current version: ${OLD_STATE}); last log lines: ${NEW_OUT}"
  fi
done < "${WORK}/new.plans"

exit 0
