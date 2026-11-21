#!/usr/bin/env bash
# Scrape the exporter against one or more Community server versions.
#
# APE_IT_SERVER_VERSIONS  space-separated image tags. Default is 7.2.0.11,
#                         the same server the OpenTelemetry suite uses.
#                         The nightly workflow passes one tag per job.
set -euo pipefail

cd "$(dirname "$0")"

versions="${APE_IT_SERVER_VERSIONS:-7.2.0.11}"
status=0
for version in $versions; do
	echo "scrape sanity against aerospike/aerospike-server:${version}"
	# 5.7's template puts "file" inside "storage-engine memory" when
	# DATA_IN_MEMORY is true, and that server rejects the parameter.
	# The file-backed device namespace is the config that server accepts.
	data_in_memory=true
	if [[ ${version} == 5.7.* ]]; then
		data_in_memory=false
	fi
	project="ape-ver-$(printf '%s' "$version" | tr '.' '-')"
	if ! AEROSPIKE_VERSION="$version" DATA_IN_MEMORY="$data_in_memory" docker compose -p "$project" up --abort-on-container-exit --exit-code-from tester; then
		status=1
	fi
	AEROSPIKE_VERSION="$version" DATA_IN_MEMORY="$data_in_memory" docker compose -p "$project" down -v >/dev/null 2>&1 || true
done
exit "$status"
