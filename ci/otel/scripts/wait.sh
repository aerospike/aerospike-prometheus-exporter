#!/bin/sh
# Checks what the OpenTelemetry exporter sends to the collector. The suite runs in two phases around a server
# restart, see ci/common/run-restart-suite.sh. The phase is the first argument.
#
# before: the server has two namespaces, test and bar.
#   1. node_up and node_stats.cluster_size arrive, and so do the series of both namespaces.
#      Both namespaces have a flash index, so index-pressure series of both arrive too. index-pressure is fetched
#      every minute here, and the series must stay in the exports between two fetches.
#   2. A series stops reporting once the server no longer returns it. A secondary index is created and dropped
#      on the server, which adds and removes the sindex stats. While the index exists its stats must be in the
#      newest exports, after it is dropped they must be gone, and when it is created again they must return.
#      node_stats.cluster_size is a series that stays, and must be in the newest exports the whole time.
#
# after: the server was restarted without the bar namespace, while the exporter kept running.
#   3. The series of bar stop, and the series of test and cluster_size keep reporting.
#   4. The index-pressure series of bar stop once a fetch no longer lists bar, and those of test keep reporting.
#
# The collector file exporter writes one JSON document per line for each push.
set -eu

output=/output/metrics.json
deadline=120
window=4
sindex_metric='aerospike.server.sindex.'
steady_metric='aerospike.server.node_stats.cluster_size'
index_deadline=240
index_pressure_metric='aerospike.server.namespace.index_pressure_total_memory'
phase=${1:-before}

# the attribute that the collector writes for a series of one namespace
ns_series() {
	printf '"key":"ns","value":{"stringValue":"%s"}' "$1"
}

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

# index_pressure_in_latest NAMESPACE: the newest exports have an index-pressure series of that namespace.
# Each export is one line, and the metrics in it are split apart to look at the one metric only.
index_pressure_in_latest() {
	[ -s "$output" ] && tail -n "$window" "$output" | sed 's/{"name":"/\n&/g' |
		grep "^{\"name\":\"${index_pressure_metric}\"" | grep -q "$(ns_series "$1")"
}

# wait_index_pressure TEST BAR: the index-pressure series of test and of bar are present or absent, as wanted,
# in the newest exports, which also need a live exporter.
wait_index_pressure() {
	i=0

	while [ "$i" -lt "$index_deadline" ]; do
		if index_pressure_in_latest test; then t=present; else t=absent; fi
		if index_pressure_in_latest bar; then b=present; else b=absent; fi

		if [ "$t" = "$1" ] && [ "$b" = "$2" ] && in_latest 'aerospike.server.node_up'; then
			return 0
		fi

		i=$((i + 1))
		sleep 1
	done

	dump_and_fail "timed out waiting for index-pressure series: test ${1}, bar ${2}"
}

asinfo_run() {
	asinfo -h aerospike -p 3000 -v "$1" >/dev/null
}

before_phase() {
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

	wait_for present "$(ns_series test)" "series of the test namespace"
	wait_for present "$(ns_series bar)" "series of the bar namespace"
	echo "series of both namespaces reported"

	# index-pressure comes with the first fetch, a minute after the exporter starts, and then stays between fetches
	wait_index_pressure present present
	echo "index-pressure series of both namespaces reported"

	n=0
	while [ "$n" -lt 5 ]; do
		index_pressure_in_latest test || dump_and_fail "index-pressure series of test dropped out between two fetches"
		n=$((n + 1))
		sleep 1
	done
	echo "index-pressure series stay between fetches"

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
}

# ns_in_latest NAMESPACE: the newest exports have a series of that namespace other than index-pressure, which is
# checked on its own because it follows the fetch interval.
ns_in_latest() {
	[ -s "$output" ] && tail -n "$window" "$output" | sed 's/{"name":"/\n&/g' |
		grep -v '^{"name":"aerospike.server.namespace.index_pressure_' | grep -q "$(ns_series "$1")"
}

# wait_namespaces KEPT GONE: the newest exports have series of KEPT and none of GONE. Both must hold in the same
# window, which also needs a live exporter, so the empty exports of the restart do not count as gone.
wait_namespaces() {
	kept=$1
	gone=$2
	i=0

	while [ "$i" -lt "$deadline" ]; do
		if ns_in_latest "$kept" && ! ns_in_latest "$gone" && in_latest "$steady_metric"; then
			return 0
		fi

		i=$((i + 1))
		sleep 1
	done

	dump_and_fail "timed out waiting for series of ${kept} without series of ${gone}"
}

after_phase() {
	# 3. the exporter reconnects to the restarted server, and what the server no longer has stops reporting
	wait_namespaces test bar
	echo "series of the removed namespace stopped, series of the kept namespace and cluster_size still report"

	# 4. index-pressure is fetched only every minute, so these series of bar go with the first fetch after the restart
	wait_index_pressure present absent
	echo "index-pressure series of the removed namespace stopped, those of the kept namespace still report"
}

case "$phase" in
before) before_phase ;;
after) after_phase ;;
*) dump_and_fail "unknown phase: ${phase}" ;;
esac

exit 0
