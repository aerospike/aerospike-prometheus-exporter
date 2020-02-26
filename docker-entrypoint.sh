#!/bin/sh
set -e

export AGENT_UPDATE_INTERVAL=${AGENT_UPDATE_INTERVAL:-5}
export AGENT_CERT_FILE=${AGENT_CERT_FILE:-""}
export AGENT_KEY_FILE=${AGENT_KEY_FILE:-""}
export AGENT_TAGS=${AGENT_TAGS:-'"agent","aerospike"'}
export AGENT_BIND_HOST=${AGENT_BIND_HOST:-""}
export AGENT_BIND_PORT=${AGENT_BIND_PORT:-9145}
export AGENT_TIMEOUT=${AGENT_TIMEOUT:-10}
export AGENT_LOG_LEVEL=${AGENT_LOG_LEVEL:-"info"}
export AS_HOST=${AS_HOST:-""}
export AS_PORT=${AS_PORT:-3000}
export AS_CERT_FILE=${AS_CERT_FILE:-""}
export AS_KEY_FILE=${AS_KEY_FILE:-""}
export AS_NODE_TLS_NAME=${AS_NODE_TLS_NAME:-""}
export AS_ROOT_CA=${AS_ROOT_CA:-""}
export AS_AUTH_MODE=${AS_AUTH_MODE:-""}
export AS_AUTH_USER=${AS_AUTH_USER:-""}
export TICKER_INTERVAL=${TICKER_INTERVAL:-5}
export TICKER_TIMEOUT=${TICKER_TIMEOUT:-5}

if [ -f /etc/aerospike-prometheus-exporter/ape.toml.template ]; then
        envsubst < /etc/aerospike-prometheus-exporter/ape.toml.template > /etc/aerospike-prometheus-exporter/ape.toml
fi

if [ "${1:0:1}" = '-' ]; then
	set -- aerospike-prometheus-exporter "$@"
fi

exec "$@"