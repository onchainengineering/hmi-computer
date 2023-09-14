#!/bin/bash
set -euo pipefail

[[ $VERBOSE == 1 ]] && set -x

# Unzip scripts and add to path.
# shellcheck disable=SC2153
echo "Extracting scaletest scripts into ${SCRIPTS_DIR}..."
base64 -d <<<"${SCRIPTS_ZIP}" >/tmp/scripts.zip
rm -rf "${SCRIPTS_DIR}" || true
mkdir -p "${SCRIPTS_DIR}"
unzip -o /tmp/scripts.zip -d "${SCRIPTS_DIR}"
rm /tmp/scripts.zip

# shellcheck disable=SC2153 source=scaletest/templates/scaletest-runner/scripts/lib.sh
. "${SCRIPTS_DIR}/lib.sh"

# Clean up any previous scaletest runs (in case /tmp persists).
rm -rf /tmp/.scaletest_* || true

# Show failure in the UI if script exits with error.
failed() {
	echo "Scaletest failed!"
	touch /tmp/.scaletest_failed
}
trap failed ERR

"${SCRIPTS_DIR}/prepare.sh"
"${SCRIPTS_DIR}/run.sh"
