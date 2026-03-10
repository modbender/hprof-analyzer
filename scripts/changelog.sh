#!/usr/bin/env bash
set -euo pipefail

# Generate CHANGELOG.md from git tags and conventional commits

OUTPUT="CHANGELOG.md"
echo "# Changelog" > "$OUTPUT"
echo "" >> "$OUTPUT"

# Get all tags sorted by version (newest first)
TAGS=$(git tag --sort=-version:refname 2>/dev/null || true)

if [ -z "$TAGS" ]; then
    echo "No tags found. Nothing to generate."
    exit 0
fi

PREV=""
for TAG in $TAGS; do
    DATE=$(git log -1 --format='%ai' "$TAG" | cut -d' ' -f1)
    echo "## ${TAG} (${DATE})" >> "$OUTPUT"
    echo "" >> "$OUTPUT"

    if [ -n "$PREV" ]; then
        RANGE="${TAG}..${PREV}"
    else
        RANGE="$TAG"
    fi

    BREAKING=""
    FEATURES=""
    FIXES=""
    OTHER=""

    while IFS= read -r msg; do
        [ -z "$msg" ] && continue
        if echo "$msg" | grep -qE '^[a-z]+(\(.+\))?!:|BREAKING CHANGE'; then
            BREAKING="${BREAKING}- ${msg}\n"
        elif echo "$msg" | grep -qE '^feat(\(.+\))?:'; then
            FEATURES="${FEATURES}- ${msg}\n"
        elif echo "$msg" | grep -qE '^fix(\(.+\))?:'; then
            FIXES="${FIXES}- ${msg}\n"
        else
            OTHER="${OTHER}- ${msg}\n"
        fi
    done < <(git log "$RANGE" --pretty=format:"%s" 2>/dev/null)

    if [ -n "$BREAKING" ]; then
        echo "### Breaking Changes" >> "$OUTPUT"
        echo -e "$BREAKING" >> "$OUTPUT"
    fi
    if [ -n "$FEATURES" ]; then
        echo "### Features" >> "$OUTPUT"
        echo -e "$FEATURES" >> "$OUTPUT"
    fi
    if [ -n "$FIXES" ]; then
        echo "### Bug Fixes" >> "$OUTPUT"
        echo -e "$FIXES" >> "$OUTPUT"
    fi
    if [ -n "$OTHER" ]; then
        echo "### Other" >> "$OUTPUT"
        echo -e "$OTHER" >> "$OUTPUT"
    fi

    PREV="$TAG"
done

echo "Generated $OUTPUT"
