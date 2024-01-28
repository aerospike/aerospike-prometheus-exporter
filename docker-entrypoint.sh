#!/bin/sh
set -e

export AGENT_CERT_FILE=${AGENT_CERT_FILE:-""}
export AGENT_KEY_FILE=${AGENT_KEY_FILE:-""}
export AGENT_ROOT_CA=${AGENT_ROOT_CA:-""}
export AGENT_KEY_FILE_PASSPHRASE=${AGENT_KEY_FILE_PASSPHRASE:-""}
export METRIC_LABELS=${METRIC_LABELS:-""}
export AGENT_BIND_HOST=${AGENT_BIND_HOST:-""}
export AGENT_BIND_PORT=${AGENT_BIND_PORT:-9145}
export AGENT_TIMEOUT=${AGENT_TIMEOUT:-10}
export AGENT_LOG_FILE=${AGENT_LOG_FILE:-""}
export AGENT_LOG_LEVEL=${AGENT_LOG_LEVEL:-""}
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
export LATENCY_BUCKETS_COUNT=${LATENCY_BUCKETS_COUNT:-0}
export NAMESPACE_METRICS_ALLOWLIST=${NAMESPACE_METRICS_ALLOWLIST:-""}
export SET_METRICS_ALLOWLIST=${SET_METRICS_ALLOWLIST:-""}
export NODE_METRICS_ALLOWLIST=${NODE_METRICS_ALLOWLIST:-""}
export XDR_METRICS_ALLOWLIST=${XDR_METRICS_ALLOWLIST:-""}
export NAMESPACE_METRICS_BLOCKLIST=${NAMESPACE_METRICS_BLOCKLIST:-""}
export SET_METRICS_BLOCKLIST=${SET_METRICS_BLOCKLIST:-""}
export NODE_METRICS_BLOCKLIST=${NODE_METRICS_BLOCKLIST:-""}
export XDR_METRICS_BLOCKLIST=${XDR_METRICS_BLOCKLIST:-""}
export USER_METRICS_USERS_ALLOWLIST=${USER_METRICS_USERS_ALLOWLIST:-""}
export USER_METRICS_USERS_BLOCKLIST=${USER_METRICS_USERS_BLOCKLIST:-""}
export JOB_METRICS_ALLOWLIST=${JOB_METRICS_ALLOWLIST:-""}
export JOB_METRICS_BLOCKLIST=${JOB_METRICS_BLOCKLIST:-""}
export SINDEX_METRICS_ALLOWLIST=${SINDEX_METRICS_ALLOWLIST:-""}
export SINDEX_METRICS_BLOCKLIST=${SINDEX_METRICS_BLOCKLIST:-""}

export AGENT_REFRESH_SYSTEM_STATS=${AGENT_OTEL:-"false"}

export AGENT_PROMETHEUS=${AGENT_PROMETHEUS:-"true"}
export AGENT_OTEL=${AGENT_OTEL:-"false"}

export USE_MOCK_DATASOURCE=${USE_MOCK_DATASOURCE:-"true"}
export AGENT_OTEL_APP_SERVICE_NAME=${AGENT_OTEL_APP_SERVICE_NAME:-"aerospike-server-metrics"}
export AGENT_OTEL_ENDPOINT=${AGENT_OTEL_ENDPOINT:-""}
export AGENT_OTEL_TLS_ENABLED=${AGENT_OTEL_TLS_ENABLED:-"true"}
export AGENT_OTEL_HEADERS=${AGENT_OTEL_HEADERS:-""}
export AGENT_OTEL_SERVER_STAT_FETCH_INTERVAL=${AGENT_OTEL_SERVER_STAT_FETCH_INTERVAL:-"60"}


if [ -f /etc/aerospike-prometheus-exporter/ape.toml.template ]; then
        env | while IFS= read -r line; do
                name=${line%%=*}
                value=${line#*=}
                if [ -n "$value" ]; then
                        sed -i --regex "s/# (.*\{$name\}.*)/\1/" /etc/aerospike-prometheus-exporter/ape.toml.template
                fi
        done
        envsubst < /etc/aerospike-prometheus-exporter/ape.toml.template > /etc/aerospike-prometheus-exporter/ape.toml
fi

if [ "${1:0:1}" = '-' ]; then
        set -- aerospike-prometheus-exporter "$@"
fi

exec "$@"