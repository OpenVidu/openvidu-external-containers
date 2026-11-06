#!/usr/bin/env bash
# Print the next free build revision N for <repository>:<version>-rN on
# Docker Hub: 0 when no revision of that version was published yet, otherwise
# the highest published N plus one.
#
# Usage: next-revision.sh <repository> <version>
# Example: next-revision.sh openvidu/mongodb 8.0.29
set -euo pipefail

repository=${1:?usage: next-revision.sh <repository> <version>}
version=${2:?usage: next-revision.sh <repository> <version>}

url="https://hub.docker.com/v2/repositories/${repository}/tags?page_size=100&name=$(jq -rn --arg v "${version}-r" '$v|@uri')"
highest=-1
while [[ -n "$url" ]]; do
  page=$(curl -fsSL --retry 3 --retry-all-errors "$url")
  page_highest=$(jq -r --arg prefix "${version}-r" '
    [.results[].name
     | select(startswith($prefix))
     | ltrimstr($prefix)
     | select(test("^[0-9]+$"))
     | tonumber]
    | max // -1' <<<"$page")
  if (( page_highest > highest )); then
    highest=$page_highest
  fi
  url=$(jq -r '.next // empty' <<<"$page")
done

echo $((highest + 1))
