#!/bin/sh
set -e

export AGENT_CERT_FILE=${AGENT_CERT_FILE:-""}
export AGENT_KEY_FILE=${AGENT_KEY_FILE:-""}
export METRIC_LABELS=${METRIC_LABELS:-""}
export AGENT_BIND_HOST=${AGENT_BIND_HOST:-""}
export AGENT_BIND_PORT=${AGENT_BIND_PORT:-9145}
export AGENT_TIMEOUT=${AGENT_TIMEOUT:-10}
export AGENT_LOG_FILE=${AGENT_LOG_FILE:-""}
export AGENT_LOG_LEVEL=${AGENT_LOG_LEVEL:-"info"}
export BASIC_AUTH_USERNAME=${BASIC_AUTH_USERNAME:-""}
export BASIC_AUTH_PASSWORD=${BASIC_AUTH_PASSWORD:-""}
export AS_HOST=${AS_HOST:-""}
export AS_PORT=${AS_PORT:-3000}
export AS_CERT_FILE=${AS_CERT_FILE:-""}
export AS_KEY_FILE=${AS_KEY_FILE:-""}
export AS_KEY_FILE_PASSPHRASE=${AS_KEY_FILE_PASSPHRASE:-""}
export AS_NODE_TLS_NAME=${AS_NODE_TLS_NAME:-""}
export AS_ROOT_CA=${AS_ROOT_CA:-""}
export AS_AUTH_MODE=${AS_AUTH_MODE:-""}
export AS_AUTH_USER=${AS_AUTH_USER:-""}
export AS_AUTH_PASSWORD=${AS_AUTH_PASSWORD:-""}
export TICKER_TIMEOUT=${TICKER_TIMEOUT:-5}
export NAMESPACE_METRICS_ALLOWLIST=${NAMESPACE_METRICS_ALLOWLIST:-""}
export SET_METRICS_ALLOWLIST=${SET_METRICS_ALLOWLIST:-""}
export NODE_METRICS_ALLOWLIST=${NODE_METRICS_ALLOWLIST:-""}
export XDR_METRICS_ALLOWLIST=${XDR_METRICS_ALLOWLIST:-""}
export NAMESPACE_METRICS_BLOCKLIST=${NAMESPACE_METRICS_BLOCKLIST:-""}
export SET_METRICS_BLOCKLIST=${SET_METRICS_BLOCKLIST:-""}
export NODE_METRICS_BLOCKLIST=${NODE_METRICS_BLOCKLIST:-""}
export XDR_METRICS_BLOCKLIST=${XDR_METRICS_BLOCKLIST:-""}

if [ -f /etc/aerospike-prometheus-exporter/ape.toml.template ]; then
        envsubst < /etc/aerospike-prometheus-exporter/ape.toml.template > /etc/aerospike-prometheus-exporter/ape.toml
fi

if [ "${1:0:1}" = '-' ]; then
	set -- aerospike-prometheus-exporter "$@"
fi

exec "$@"