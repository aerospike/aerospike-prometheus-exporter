#!/usr/bin/env bash
# Build the release image, check that docker-entrypoint.sh writes a Secret
# Agent setting or a backup and restore metric filter only when its environment
# variable is set, then start that image against a Community server and scrape
# /metrics.
set -euo pipefail

root=$(cd "$(dirname "$0")/../.." && pwd)
image=${APE_IT_DOCKER_IMAGE:-ape-it-docker}
config=/etc/aerospike-prometheus-exporter/ape.toml

cd "$root"
docker build -t "$image" .

render() {
	docker run --rm "$@" "$image" cat "$config"
}

section() {
	awk '/^\[Agent\.SecretAgent\]/{found=1} found{print} /^\[Agent\.OpenTelemetry\]/{if(found) exit}'
}

fail() {
	echo "$1" >&2
	echo "----- rendered [Agent.SecretAgent] -----" >&2
	printf '%s\n' "$2" | section >&2
	echo "----- rendered metric filters -----" >&2
	printf '%s\n' "$2" | grep -E '_metrics_(allow|block)list' >&2 || true
	exit 1
}

expect_active() {
	local rendered=$1 line=$2
	printf '%s\n' "$rendered" | grep -Eq "^[[:space:]]*${line}$" || fail "missing active line: ${line}" "$rendered"
}

expect_commented() {
	local rendered=$1 key=$2
	printf '%s\n' "$rendered" | grep -Eq "^[[:space:]]*# ${key} =" || fail "expected ${key} to stay commented" "$rendered"
}

echo "tcp: address and port are set"
tcp=$(render -e AGENT_SECRET_AGENT_ADDRESS=127.0.0.1 -e AGENT_SECRET_AGENT_PORT=3005)
expect_active "$tcp" 'address = "127.0.0.1"'
expect_active "$tcp" 'port = 3005'
for key in socket_path tls_name ca_file cert_file key_file key_file_passphrase timeout; do
	expect_commented "$tcp" "$key"
done

echo "default: no Secret Agent variables"
plain=$(render)
for key in address port socket_path tls_name ca_file cert_file key_file key_file_passphrase timeout; do
	expect_commented "$plain" "$key"
done

echo "uds: socket_path is set"
uds=$(render -e AGENT_SECRET_AGENT_SOCKET_PATH=/tmp/secret-agent.sock)
expect_active "$uds" 'socket_path = "/tmp/secret-agent.sock"'
for key in address port tls_name ca_file cert_file key_file key_file_passphrase timeout; do
	expect_commented "$uds" "$key"
done

echo "Docker Secret Agent config rendering passed"

# The backup and restore filters are set through environment variables like the other metric filters.
# A value is written as a list of quoted names, and the line stays commented out when the variable is unset.
echo "filters: default, no backup or restore filter variables"
for key in backup_metrics_allowlist backup_metrics_blocklist restore_metrics_allowlist restore_metrics_blocklist; do
	expect_commented "$plain" "$key"
done

echo "filters: BACKUP_METRICS_ALLOWLIST alone"
backup_only=$(render -e BACKUP_METRICS_ALLOWLIST='"progress_pct"')
expect_active "$backup_only" 'backup_metrics_allowlist = \["progress_pct"\]'
for key in backup_metrics_blocklist restore_metrics_allowlist restore_metrics_blocklist; do
	expect_commented "$backup_only" "$key"
done

echo "filters: all four backup and restore variables"
filters=$(render \
	-e BACKUP_METRICS_ALLOWLIST='"progress_pct","recs_*"' \
	-e BACKUP_METRICS_BLOCKLIST='"bytes_*"' \
	-e RESTORE_METRICS_ALLOWLIST='"progress_pct","done"' \
	-e RESTORE_METRICS_BLOCKLIST='"*"')
expect_active "$filters" 'backup_metrics_allowlist = \["progress_pct","recs_\*"\]'
expect_active "$filters" 'backup_metrics_blocklist = \["bytes_\*"\]'
expect_active "$filters" 'restore_metrics_allowlist = \["progress_pct","done"\]'
expect_active "$filters" 'restore_metrics_blocklist = \["\*"\]'
echo "Docker backup and restore filter rendering passed"

server=${APE_IT_DOCKER_SERVER:-7.2.0.11}
echo "scrape: release image against aerospike/aerospike-server:${server}"
export APE_IT_DOCKER_IMAGE="$image"
cleanup() {
	AEROSPIKE_VERSION="$server" docker compose -f ci/docker/docker-compose.yml -p ape-docker-image down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT
AEROSPIKE_VERSION="$server" docker compose -f ci/docker/docker-compose.yml -p ape-docker-image up --abort-on-container-exit --exit-code-from tester
echo "Docker release image scrape passed"
