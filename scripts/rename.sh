#!/usr/bin/env bash
# Rename the Go module after cloning the template.
#   ./scripts/rename.sh github.com/acme/my-service
set -euo pipefail

OLD="github.com/your-username/gin-template"
NEW="${1:?usage: $0 <new/module/path>}"

cd "$(dirname "$0")/.."

# Portable in-place sed (GNU and BSD/macOS).
files=$(grep -rl --exclude-dir=.git --exclude-dir=bin --exclude=rename.sh "$OLD" . || true)
for f in $files; do
  sed -i.bak "s#${OLD}#${NEW}#g" "$f" && rm -f "$f.bak"
done

# Compose project name / binary names stay generic; edit compose.yaml `name:` if you like.
echo "Renamed module to ${NEW}. Now run: go mod tidy && make test"
