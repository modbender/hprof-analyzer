#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Determine next version
NEXT_VERSION=$("$SCRIPT_DIR/bump-version.sh")
echo "Next version: $NEXT_VERSION"

# Generate changelog
"$SCRIPT_DIR/changelog.sh"

# Commit changelog if changed
if ! git diff --quiet CHANGELOG.md 2>/dev/null; then
    git add CHANGELOG.md
    git commit -m "docs: update CHANGELOG.md for ${NEXT_VERSION}"
fi

# Create annotated tag
git tag -a "$NEXT_VERSION" -m "Release ${NEXT_VERSION}"

echo "Created tag $NEXT_VERSION"
echo "Run 'git push origin main --tags' to trigger the release workflow."
