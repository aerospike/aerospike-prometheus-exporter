#!/usr/bin/env bash
# Start a secured Enterprise server, create three users, run asbench as each
# user, and check the user processor against QueryUsers.
#
# APE_IT_FEATURE_KEY_FILE  path to an Enterprise feature key. The key is mounted
#                          into the server and is not committed. When unset, the
#                          suite skips and exits 0.
set -euo pipefail

cd "$(dirname "$0")"

if [[ -z ${APE_IT_FEATURE_KEY_FILE:-} || ! -f ${APE_IT_FEATURE_KEY_FILE} ]]; then
	echo "APE_IT_FEATURE_KEY_FILE is unset; skipping the user stats suite"
	exit 0
fi

export APE_IT_FEATURE_KEY_FILE

cleanup() {
	docker compose down -v
}
trap cleanup EXIT

docker compose up --abort-on-container-exit --exit-code-from tester
