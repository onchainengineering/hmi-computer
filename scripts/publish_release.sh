#!/usr/bin/env bash

# This script generates release notes and publishes all of the given assets to
# GitHub releases. Depends on GitHub CLI.
#
# Usage: ./publish_release.sh [--version 1.2.3] [--dry-run] path/to/asset1 path/to/asset2 ...
#
# The supplied images must already be pushed to the registry or this will fail.
# Also, the source images cannot be in a different registry than the target
# image generated by ./image_tag.sh.
# The supplied assets will be uploaded to the GitHub release as-is, as well as a
# file containing checksums.
#
# If no version is specified, defaults to the version from ./version.sh. The
# script will exit early if the branch is not tagged with the provided version
# (plus the "v" prefix) unless run with --dry-run.
#
# If the --dry-run parameter is supplied, the release will not be published to
# GitHub at all.
#
# Returns the link to the created GitHub release (unless --dry-run was
# specified).

set -euo pipefail
# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

version=""
dry_run=0

args="$(getopt -o "" -l version:,dry-run -- "$@")"
eval set -- "$args"
while true; do
    case "$1" in
    --version)
        version="$2"
        shift 2
        ;;
    --dry-run)
        dry_run=1
        shift
        ;;
    --)
        shift
        break
        ;;
    *)
        error "Unrecognized option: $1"
        ;;
    esac
done

# Remove the "v" prefix.
version="${version#v}"
if [[ "$version" == "" ]]; then
    version="$(execrelative ./version.sh)"
fi

# Verify that we're currently checked out on the supplied tag.
new_tag="v$version"
if [[ "$(git describe --always)" != "$new_tag" ]]; then
    if [[ "$dry_run" == 0 ]]; then
        error "The provided version '$new_tag' does not match the current git describe output '$(git describe --always)'"
    fi

    log "The provided version does not match the current git tag, but --dry-run was supplied so continuing..."
fi

# This returns the tag before the current tag.
old_tag="$(git describe --abbrev=0 --tags "$(git rev-list --tags --skip=1 --max-count=1)")"

# Craft the release notes.
release_notes="
## Changelog

$(git log --no-merges --pretty=format:"- %h %s" "$old_tag..$new_tag")

## Container Image
- \`docker pull $(execrelative ./image_tag.sh --version "$version")\`

"

# Create temporary release folder so we can generate checksums.
temp_dir="$(mktemp -d)"
for f in "$@"; do
    ln -s "$(realpath "$f")" "$temp_dir/"
done

# Generate checksums file. sha256sum seems to play nicely with symlinks so this
# works well.
pushd "$temp_dir"
sha256sum ./* | sed -e 's/\.\///' - >"coder_${version}_checksums.txt"
popd

log "--- Creating release $new_tag"
log
log "Description:"
echo "$release_notes" | sed -e 's/^/\t/' - 1>&2
log
log "Contents:"
pushd "$temp_dir"
find ./* 2>&1 | sed -e 's/^/\t/;s/\.\///' - 1>&2
popd
log
log

# We echo the release notes in instead of writing to a file and referencing that
# to prevent GitHub CLI from becoming interactive.
#
# GitHub CLI seems to follow symlinks when uploading files.
echo "$release_notes" |
    maybedryrun "$dry_run" gh release create \
        --title "$new_tag" \
        --notes-file - \
        "$new_tag" \
        "$temp_dir"/*

rm -rf "$temp_dir"
