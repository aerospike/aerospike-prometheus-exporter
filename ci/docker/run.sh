#!/usr/bin/env bash
# Build the release image and check that docker-entrypoint.sh writes a Secret
# Agent setting only when its environment variable is set.
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
