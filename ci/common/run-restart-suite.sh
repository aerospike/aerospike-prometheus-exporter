#!/usr/bin/env bash
# Runs a compose suite in two phases around a server restart.
#
#   run-restart-suite.sh <suite dir> <service>...
#
# The services are started with a server that has two namespaces, test and bar, and the tester runs its "before"
# phase. The server is then recreated with a config that has no bar namespace, which is the only way to remove a
# namespace, while the exporter keeps running, and the tester runs its "after" phase. The tester gets the phase as
# its first argument.
set -euo pipefail

suite=${1:?suite directory, for example ci/otel}
shift

root=$(cd "$(dirname "$0")/../.." && pwd)
export CI_GO_VERSION=${CI_GO_VERSION:-$(awk '/^go /{print $2; exit}' "$root/go.mod")}
compose=(docker compose -f "$root/$suite/docker-compose.yml")

cleanup() {
	"${compose[@]}" down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT

show_logs() {
	echo "----- exporter and server logs -----" >&2
	"${compose[@]}" logs --no-color --tail 60 ape aerospike >&2 || true
}

# start ARG...: compose up and wait, printing the logs if a service fails to start
start() {
	if ! "${compose[@]}" up -d --wait "$@"; then
		show_logs
		return 1
	fi
}

run_phase() {
	echo "===== $suite: $1 phase ====="
	# --no-deps keeps compose from recreating the server, which it would do if its config differed
	if ! "${compose[@]}" run --rm -T --no-deps tester "$1"; then
		show_logs
		return 1
	fi
}

start "$@"
run_phase before

echo "===== $suite: restarting the server without the bar namespace ====="
# the compose file reads this for the server's config, so keep it set for every later compose command
export APE_IT_AEROSPIKE_CONF=aerospike-one-namespace.conf
start --force-recreate --no-deps aerospike
run_phase after

echo "$suite restart suite passed"
