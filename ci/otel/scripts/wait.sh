#!/bin/sh
set -eu

output=/output/metrics.json
deadline=120
i=0

while [ "$i" -lt "$deadline" ]; do
	if [ -s "$output" ] &&
		grep -q 'aerospike.server.node_up' "$output" &&
		grep -q 'aerospike.server.node_stats.cluster_size' "$output"; then
		echo "collector received node_up and cluster_size"
		exit 0
	fi
	i=$((i + 1))
	sleep 1
done

echo "collector did not receive OpenTelemetry metrics within ${deadline}s"
if [ -f "$output" ]; then
	echo "--- metrics file ---"
	wc -c "$output"
	tail -c 2000 "$output" || true
else
	echo "metrics file was not created"
fi
exit 1
