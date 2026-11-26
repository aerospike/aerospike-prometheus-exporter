#!/bin/sh
# Checks the Prometheus HTTP listener and what it exports.
#
#   1. /health and / answer without credentials.
#   2. /metrics answers 401 without credentials and with a wrong password, and 200 with the right ones.
#   3. node_up, namespace and node stats are exported.
#   4. A series stops reporting once the server no longer returns it. A secondary index is created and dropped
#      on the server, which adds and removes the sindex stats. While the index exists its stats must be in the
#      scrape, after it is dropped they must be gone, and when it is created again they must return.
#      node_stats stays in the scrape the whole time.
set -eu

base=http://ape:9145
user=exporter
pass=prom-it-9f3c
deadline=120

# a stalled request must not keep the polling loops from reaching their deadline
CURL='curl --connect-timeout 3 --max-time 10'
sindex_metric='^aerospike_sindex_'
steady_metric='^aerospike_node_stats_'

body=""

dump_and_fail() {
	echo "$1" >&2
	echo "--- last scrape (node_up, sindex and node_stats series) ---" >&2
	printf '%s\n' "$body" | grep -E 'aerospike_node_up|aerospike_sindex_|aerospike_node_stats_' | head -n 30 >&2 || true
	exit 1
}

status_of() {
	$CURL -s -o /dev/null -w '%{http_code}' "$@"
}

scrape() {
	body=$($CURL -fsS -u "${user}:${pass}" "${base}/metrics" 2>/dev/null) || body=""
}

# wait_for present|absent PATTERN DESCRIPTION: the scrape of /metrics matches, or does not match, PATTERN.
# An empty or failed scrape does not count as absent, the node must still be reported up.
wait_for() {
	want=$1
	pattern=$2
	description=$3
	i=0

	while [ "$i" -lt "$deadline" ]; do
		scrape
		if printf '%s\n' "$body" | grep '^aerospike_node_up{' | grep -q ' 1$'; then
			if printf '%s\n' "$body" | grep -q "$pattern"; then
				seen=present
			else
				seen=absent
			fi
			if [ "$seen" = "$want" ]; then
				return 0
			fi
		fi
		i=$((i + 1))
		sleep 1
	done

	dump_and_fail "timed out waiting for: ${description}"
}

asinfo_run() {
	asinfo -h aerospike -p 3000 -v "$1" >/dev/null
}

# 1. the listener is up, and /health and / need no credentials
i=0
while [ "$i" -lt "$deadline" ]; do
	if [ "$(status_of "${base}/health")" = "200" ]; then
		break
	fi
	i=$((i + 1))
	sleep 1
done
if [ "$i" -ge "$deadline" ]; then
	dump_and_fail "the exporter did not answer /health within ${deadline}s"
fi

[ "$($CURL -s "${base}/health")" = "OK" ] || dump_and_fail "/health did not answer OK"
$CURL -s "${base}/" | grep -q 'Aerospike Prometheus Exporter' || dump_and_fail "/ did not show the exporter page"
echo "/health and / answer without credentials"

# 2. /metrics needs the basic auth credentials
[ "$(status_of "${base}/metrics")" = "401" ] || dump_and_fail "/metrics without credentials was not 401"
[ "$(status_of -u "${user}:wrong" "${base}/metrics")" = "401" ] || dump_and_fail "/metrics with a wrong password was not 401"
[ "$(status_of -u "other:${pass}" "${base}/metrics")" = "401" ] || dump_and_fail "/metrics with a wrong user was not 401"
$CURL -s -D - -o /dev/null "${base}/metrics" | grep -qi '^WWW-Authenticate: Basic realm=' || dump_and_fail "401 did not carry WWW-Authenticate"
echo "/metrics answers 401 without the right credentials"

# 3. the exporter reports a live node
wait_for present '^aerospike_namespace_' "a live node with namespace stats"
printf '%s\n' "$body" | grep -q "$steady_metric" || dump_and_fail "node stats are missing from the scrape"
echo "/metrics answers 200 with credentials and shows node_up, namespace and node stats"

# 4. a series that the server stops returning stops reporting
index='sindex-create:ns=test;set=ape_it;indexname=ape_it_idx;indexdata=age,numeric'
drop='sindex-delete:ns=test;set=ape_it;indexname=ape_it_idx'

wait_for absent "$sindex_metric" "no sindex series before the index exists"

asinfo_run "$index"
wait_for present "$sindex_metric" "sindex series after the index is created"
printf '%s\n' "$body" | grep -q "$steady_metric" || dump_and_fail "node stats are missing while the index exists"
echo "sindex series reported while the index exists"

asinfo_run "$drop"
wait_for absent "$sindex_metric" "sindex series to stop after the index is dropped"
printf '%s\n' "$body" | grep -q "$steady_metric" || dump_and_fail "node stats stopped when the index was dropped"
echo "sindex series stopped after the index was dropped, node stats still report"

asinfo_run "$index"
wait_for present "$sindex_metric" "sindex series to return after the index is created again"
echo "sindex series reported again after the index was created again"

asinfo_run "$drop"
exit 0
