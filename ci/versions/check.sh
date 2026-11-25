#!/bin/sh
# Wait until the exporter scrape shows a live node and both processor families.
set -eu

body=""
i=0
while [ "$i" -lt 45 ]; do
	if body=$(wget -qO- http://ape:9145/metrics 2>/dev/null); then
		if printf '%s\n' "$body" | grep '^aerospike_node_up{' | grep -q ' 1$' \
			&& printf '%s\n' "$body" | grep -q '^aerospike_namespace_' \
			&& printf '%s\n' "$body" | grep -q '^aerospike_node_stats_'; then
			exit 0
		fi
	fi
	i=$((i + 1))
	sleep 2
done

echo "exporter scrape did not show node_up 1 with namespace and node stats" >&2
printf '%s\n' "$body" | grep -E 'aerospike_node_up|aerospike_namespace_|aerospike_node_stats_' | head -n 20 >&2 || true
exit 1
