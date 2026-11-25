#!/bin/sh
# Checks what the OpenTelemetry exporter sends to the collector.
#
#   1. node_up and node_stats.cluster_size arrive.
#   2. A series stops reporting once the server no longer returns it. A secondary index is created and dropped
#      on the server, which adds and removes the sindex stats. While the index exists its stats must be in the
#      newest exports, after it is dropped they must be gone, and when it is created again they must return.
#      node_stats.cluster_size is a series that stays, and must be in the newest exports the whole time.
#
# The collector file exporter writes one JSON document per line for each push.
set -eu

output=/output/metrics.json
deadline=120
window=4
sindex_metric='aerospike.server.sindex.'
steady_metric='aerospike.server.node_stats.cluster_size'

dump_and_fail() {
	echo "$1"
	if [ -f "$output" ]; then
		echo "--- metrics file ---"
		wc -c "$output"
		echo "--- sindex series in the newest $window exports ---"
		tail -n "$window" "$output" | grep -o "${sindex_metric}[a-z_]*" | sort -u || true
		echo "--- tail ---"
		tail -c 2000 "$output" || true
	else
		echo "metrics file was not created"
	fi
	exit 1
}

# in_latest PATTERN: the newest exports contain PATTERN
in_latest() {
	[ -s "$output" ] && tail -n "$window" "$output" | grep -q "$1"
}

# wait_for present|absent PATTERN DESCRIPTION
wait_for() {
	want=$1
	pattern=$2
	description=$3
	i=0

	while [ "$i" -lt "$deadline" ]; do
		if in_latest "$pattern"; then
			seen=present
		else
			seen=absent
		fi

		# the exporter must still be pushing, so an empty window does not count as absent
		if [ "$seen" = "$want" ] && in_latest 'aerospike.server.node_up'; then
			return 0
		fi

		i=$((i + 1))
		sleep 1
	done

	dump_and_fail "timed out waiting for: ${description}"
}

asinfo_run() {
	asinfo -h aerospike -p 3000 -v "$1" >/dev/null
}

# 1. the exporter is up
i=0
while [ "$i" -lt "$deadline" ]; do
	if [ -s "$output" ] &&
		grep -q 'aerospike.server.node_up' "$output" &&
		grep -q 'aerospike.server.node_stats.cluster_size' "$output"; then
		echo "collector received node_up and cluster_size"
		break
	fi
	i=$((i + 1))
	sleep 1
done

if [ "$i" -ge "$deadline" ]; then
	dump_and_fail "collector did not receive OpenTelemetry metrics within ${deadline}s"
fi

# 2. a series that the server stops returning stops reporting
index='sindex-create:ns=test;set=ape_it;indexname=ape_it_idx;indexdata=age,numeric'
drop='sindex-delete:ns=test;set=ape_it;indexname=ape_it_idx'

wait_for absent "$sindex_metric" "no sindex series before the index exists"

asinfo_run "$index"
wait_for present "$sindex_metric" "sindex series after the index is created"
in_latest "$steady_metric" || dump_and_fail "cluster_size is missing while the index exists"
echo "sindex series reported while the index exists"

asinfo_run "$drop"
wait_for absent "$sindex_metric" "sindex series to stop after the index is dropped"
in_latest "$steady_metric" || dump_and_fail "cluster_size stopped when the index was dropped"
echo "sindex series stopped after the index was dropped, cluster_size still reports"

asinfo_run "$index"
wait_for present "$sindex_metric" "sindex series to return after the index is created again"
echo "sindex series reported again after the index was created again"

asinfo_run "$drop"
exit 0
