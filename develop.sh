#!/usr/bin/env bash

set -euo pipefail

EMAIL="${EMAIL:-admin@coder.com}"
USERNAME="${USERNAME:-admin}"
ORGANIZATION="${ORGANIZATION:-ACME-Corp}"
PASSWORD="${PASSWORD:-password}"
PROJECT_ROOT="$(git rev-parse --show-toplevel)"
cd "${PROJECT_ROOT}"

# Helper to create an initial user
function create_initial_user() {
  # TODO: We need to wait for `coderd` to spin up -
  # need to replace with a deterministic strategy
  sleep 5s
  
  curl -X POST \
  -d "{\"email\": \"$EMAIL\", \"username\": \"$USERNAME\", \"organization\": \"$ORGANIZATION\", \"password\": \"$PASSWORD\"}" \
  -H 'Content-Type:application/json' \
  http://localhost:3000/api/v2/user
}

# Do initial build - a dev build for coderd.
# It's OK that we don't build the front-end before - because the front-end
# assets are handled by the `yarn dev` devserver.
make bin/coderd

# This is a way to run multiple processes in parallel, and have Ctrl-C work correctly
# to kill both at the same time. For more details, see:
# https://stackoverflow.com/questions/3004811/how-do-you-run-multiple-programs-in-parallel-from-a-bash-script
(trap 'kill 0' SIGINT; create_initial_user & CODERV2_HOST=http://127.0.0.1:3000 yarn dev & ./bin/coderd)