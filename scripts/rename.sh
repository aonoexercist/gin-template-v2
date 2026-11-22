#!/usr/bin/env bash
# Rename the Go module and compose project name after cloning the template.
#   ./scripts/rename.sh github.com/myuser/my-api
set -euo pipefail

OLD="github.com/aonoexercist/gin-template-v2"
NEW="${1:?usage: $0 <new/module/path>}"

cd "$(dirname "$0")/.."

# Derive the project name from the last path segment.
# Skip a Go major-version suffix (e.g. github.com/acme/my-api/v2 -> my-api).
NAME="${NEW##*/}"
if [[ "$NAME" =~ ^v[0-9]+$ && "$NEW" == */*/* ]]; then
  trimmed="${NEW%/*}"
  NAME="${trimmed##*/}"
fi
# Compose project names must be lowercase [a-z0-9_-] and start with a letter/digit.
NAME=$(printf '%s' "$NAME" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9_-]+/-/g; s/^[^a-z0-9]+//')
[ -n "$NAME" ] || { echo "could not derive a valid project name from '$NEW'" >&2; exit 1; }

# Portable in-place sed (GNU and BSD/macOS).
files=$(grep -rl --exclude-dir=.git --exclude-dir=bin --exclude=rename.sh "$OLD" . || true)
for f in $files; do
  sed -i.bak "s#${OLD}#${NEW}#g" "$f" && rm -f "$f.bak"
done

# Update the compose project name (compose.yaml or compose.yml).
for c in compose.yaml compose.yml; do
  if [ -f "$c" ]; then
    sed -i.bak -E "s#^name:.*#name: ${NAME}#" "$c" && rm -f "$c.bak"
    echo "Set ${c} name to: ${NAME}"
  fi
done

echo "Renamed module to ${NEW}. Now run: go mod tidy && make test"