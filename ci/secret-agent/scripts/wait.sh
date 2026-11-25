#!/bin/sh
set -eu

apk add --no-cache netcat-openbsd >/dev/null

i=0
while [ "$i" -lt 60 ]; do
	if nc -z sa-tcp 3005 \
		&& nc -z sa-tls 3005 \
		&& nc -z sa-mtls 3005 \
		&& nc -z sa-tcp-b64 3005 \
		&& [ -S /run/secret-agent.sock ]; then
		exit 0
	fi
	i=$((i + 1))
	sleep 1
done

echo "secret agent listeners were not ready" >&2
exit 1
