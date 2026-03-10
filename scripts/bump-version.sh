#!/usr/bin/env bash
set -euo pipefail

# Get last tag, default to v0.0.0
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Parse major.minor.patch
VERSION="${LAST_TAG#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

# Determine bump type from conventional commits since last tag
BUMP="patch"
while IFS= read -r line; do
    # Breaking change: major bump
    if echo "$line" | grep -qE '^[a-z]+(\(.+\))?!:|BREAKING CHANGE'; then
        BUMP="major"
        break
    fi
    # feat: minor bump (don't override major)
    if echo "$line" | grep -qE '^feat(\(.+\))?:'; then
        BUMP="minor"
    fi
done < <(git log "${LAST_TAG}..HEAD" --pretty=format:"%s" 2>/dev/null || git log --pretty=format:"%s")

case "$BUMP" in
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
    minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
    patch) PATCH=$((PATCH + 1)) ;;
esac

echo "v${MAJOR}.${MINOR}.${PATCH}"
