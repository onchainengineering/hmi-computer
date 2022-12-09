#!/usr/bin/env bash

# This script generates release notes and publishes all of the given assets to
# GitHub releases. Depends on GitHub CLI.
#
# THIS IS NOT INTENDED TO BE CALLED BY DEVELOPERS! This is called by the release
# pipeline to do the final publish step. If you want to create a release use:
#   git tag -a -m "$ver" "$ver" && git push origin "$ver"
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
# shellcheck source=scripts/lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

if [[ "${CI:-}" == "" ]]; then
	error "This script must be run in CI"
fi

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

# Check dependencies
dependencies gh

# Remove the "v" prefix.
version="${version#v}"
if [[ "$version" == "" ]]; then
	version="$(execrelative ./version.sh)"
fi

# realpath-ify all input files so we can cdroot below.
files=()
for f in "$@"; do
	if [[ ! -e "$f" ]]; then
		error "File not found: $f"
	fi
	files+=("$(realpath "$f")")
done
if [[ "${#files[@]}" == 0 ]]; then
	error "No files supplied"
fi

if [[ "$dry_run" == 0 ]] && [[ "$version" == *dev* ]]; then
	error "Cannot publish a dev version to GitHub"
fi

# The git commands need to be executed from within the repository.
cdroot

# Verify that we're currently checked out on the supplied tag.
new_tag="v$version"
if [[ "$(git describe --always)" != "$new_tag" ]]; then
	if [[ "$dry_run" == 0 ]]; then
		error "The provided version '$new_tag' does not match the current git describe output '$(git describe --always)'"
	fi

	log "The provided version does not match the current git tag, but --dry-run was supplied so continuing..."
fi

# This returns the tag before the current tag.
old_tag="$(git describe --abbrev=0 HEAD^1)"

# For dry-run builds we want to use the SHA instead of the tag, because the new
# tag probably doesn't exist.
new_ref="$new_tag"
if [[ "$dry_run" == 1 ]]; then
	new_ref="$(git rev-parse --short HEAD)"
fi

# shellcheck source=scripts/check_commit_metadata.sh
source "$SCRIPT_DIR/check_commit_metadata.sh" "$old_tag..$new_ref"

# Craft the release notes.
release_notes="$(execrelative ./generate_release_notes.sh --old-version "$old_tag" --new-version "$new_tag" --ref "$new_ref")"

release_notes_file="$(mktemp)"
echo "$release_notes" >"$release_notes_file"

# Create temporary release folder so we can generate checksums. Both the
# sha256sum and gh binaries support symlinks as input files so this works well.
temp_dir="$(mktemp -d)"
for f in "${files[@]}"; do
	ln -s "$f" "$temp_dir/"
done

# Generate checksums file which will be uploaded to the GitHub release.
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

# We pipe `true` into `gh` so that it never tries to be interactive.
true |
	maybedryrun "$dry_run" gh release create \
		--title "$new_tag" \
		--notes-file "$release_notes_file" \
		"$new_tag" \
		"$temp_dir"/*

rm -rf "$temp_dir"
rm -rf "$release_notes_file"
