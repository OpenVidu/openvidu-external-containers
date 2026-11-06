#!/usr/bin/env bash
# Print, one per line, the images whose build inputs changed between two
# commits: files copied into the image or its values in versions.env.
# Changes to READMEs, tests or compose examples do not trigger a rebuild.
#
# Usage: changed-images.sh <base-commit> <head-commit>
# When the base commit is missing (first push of a branch, force push), every
# image is printed.
set -euo pipefail

base=${1:-}
head=${2:?usage: changed-images.sh <base-commit> <head-commit>}
images=(loki mimir minio mc mongo)

if [[ -z "$base" || "$base" =~ ^0+$ ]] || ! git cat-file -e "${base}^{commit}" 2>/dev/null; then
  printf '%s\n' "${images[@]}"
  exit 0
fi

declare -A changed=()
mark() { for image in "$@"; do changed[$image]=1; done; }

while IFS= read -r file; do
  case "$file" in
    images/mc/*_test.go) ;;
    # The minio image embeds the mc client built from images/mc.
    images/mc/Dockerfile) mark mc ;;
    images/mc/go.mod | images/mc/go.sum | images/mc/*.go) mark mc minio ;;
    images/minio/Dockerfile | images/minio/prebuildfs/* | images/minio/rootfs/*) mark minio ;;
    images/mongo/Dockerfile | images/mongo/install.sh | images/mongo/prebuildfs/* | images/mongo/rootfs/*) mark mongo ;;
    images/loki/Dockerfile) mark loki ;;
    images/mimir/Dockerfile) mark mimir ;;
  esac
done < <(git diff --name-only "$base" "$head")

value() { git show "$1:versions.env" 2>/dev/null | sed -n "s/^$2=//p"; }
check() {
  local image=$1 var
  shift
  for var in "$@"; do
    if [[ "$(value "$base" "$var")" != "$(value "$head" "$var")" ]]; then
      mark "$image"
      return
    fi
  done
}
check loki LOKI_TAG LOKI_DIGEST
check mimir MIMIR_TAG MIMIR_DIGEST
check minio MINIO_TAG
check mongo MONGODB_MAJOR_MINOR MONGODB_VERSION MONGOSH_VERSION YQ_VERSION WAIT_FOR_PORT_COMMIT RENDER_TEMPLATE_COMMIT

for image in "${images[@]}"; do
  if [[ -n "${changed[$image]:-}" ]]; then
    echo "$image"
  fi
done
