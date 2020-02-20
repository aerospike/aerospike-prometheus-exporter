FROM golang:alpine AS builder


ADD . $GOPATH/src/github.com/citrusleaf/aerospike-prometheus-exporter
WORKDIR $GOPATH/src/github.com/citrusleaf/aerospike-prometheus-exporter
RUN apk add git\
  && apk add gettext libintl \
  && go get ./... \
  && go build -o aerospike-prometheus-exporter . \
  && cp aerospike-prometheus-exporter /aerospike-prometheus-exporter


FROM golang:alpine
ENV AGENT_UPDATE_INTERVAL=5\
    AGENT_CERT_FILE=""\
    AGENT_KEY_FILE=""\
    AGENT_TAGS='"agent","aerospike"'\
    AGENT_BIND_HOST=""\
    AGENT_BIND_PORT=9145\
    AGENT_TIMEOUT=10\
    AGENT_LOG_LEVEL="info"\
    AS_HOST=""\
    AS_PORT=3000\
    AS_CERT_FILE=""\
    AS_KEY_FILE=""\
    AS_NODE_TLS_NAME=""\
    AS_ROOT_CA=""\
    AS_AUTH_MODE=""\
    AS_AUTH_USER=""\
    TICKER_INTERVAL=5\
    TICKER_TIMEOUT=5

RUN apk add gettext libintl 
COPY --from=builder /aerospike-prometheus-exporter /usr/bin/aerospike-prometheus-exporter
COPY ape.toml.template /etc/aerospike-prometheus-exporter/ape.toml.template 
# you could change the port via env var and then would have to --expose in run.
# That is likely unnecessary though
EXPOSE 9145 

CMD cat /etc/aerospike-prometheus-exporter/ape.toml.template && env && envsubst < /etc/aerospike-prometheus-exporter/ape.toml.template > /etc/aerospike-prometheus-exporter/ape.toml && cat /etc/aerospike-prometheus-exporter/ape.toml && aerospike-prometheus-exporter -config /etc/aerospike-prometheus-exporter/ape.toml
