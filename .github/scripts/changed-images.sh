#!/usr/bin/env bash
# Print, one per line, the images whose build inputs changed between two
# commits: files copied into the image or its values in versions.env.
# Changes to READMEs, tests or compose examples do not trigger a rebuild.
# A change to the Go module of images/mc rebuilds mc (and minio, which embeds
# it) only when it changes what goes into the mc binary: its non-test sources,
# the go/toolchain lines of go.mod or the modules the binary links. Updates of
# test-only modules (testcontainers-go and its dependencies) rebuild nothing.
# That check needs Go; without it, any change to the module rebuilds both.
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

# What the mc binary is built from at a commit: its non-test sources, the
# go/toolchain lines of go.mod and the modules its packages import, test
# dependencies excluded. Fails when Go is missing or the module cannot be
# listed.
mc_fingerprint() {
  local commit=$1 dir status=0
  command -v go >/dev/null || return 1
  dir=$(mktemp -d)
  {
    git archive "$commit" images/mc | tar -x -C "$dir" &&
      git ls-tree "$commit" images/mc/ | awk '$4 ~ /\.go$/ && $4 !~ /_test\.go$/ {print $3, $4}' &&
      grep -E '^(go|toolchain)[[:space:]]' "$dir/images/mc/go.mod" &&
      (cd "$dir/images/mc" && GOFLAGS=-mod=readonly go list -deps \
        -f '{{with .Module}}{{.Path}} {{.Version}}{{with .Replace}} => {{.Path}} {{.Version}}{{end}}{{end}}' . |
        sort -u)
  } || status=$?
  rm -r -f -- "$dir"
  return "$status"
}
mc_module_changed=

while IFS= read -r file; do
  case "$file" in
    images/mc/*_test.go) ;;
    images/mc/Dockerfile) mark mc ;;
    # The minio image embeds the mc client built from images/mc, so a change
    # of the mc binary rebuilds both (decided after the loop).
    images/mc/go.mod | images/mc/go.sum | images/mc/*.go) mc_module_changed=1 ;;
    images/minio/Dockerfile | images/minio/prebuildfs/* | images/minio/rootfs/*) mark minio ;;
    images/mongo/Dockerfile | images/mongo/install.sh | images/mongo/prebuildfs/* | images/mongo/rootfs/* | images/mongo/compat/*) mark mongo ;;
    images/loki/Dockerfile) mark loki ;;
    images/mimir/Dockerfile) mark mimir ;;
  esac
done < <(git diff --name-only "$base" "$head")

if [[ -n "$mc_module_changed" ]]; then
  if ! base_mc=$(mc_fingerprint "$base") || ! head_mc=$(mc_fingerprint "$head"); then
    echo "Could not compare the inputs of the mc binary: rebuilding mc and minio" >&2
    mark mc minio
  elif [[ "$base_mc" != "$head_mc" ]]; then
    mark mc minio
  else
    echo "The Go module of images/mc changed, but not the inputs of the mc binary" >&2
  fi
fi

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
