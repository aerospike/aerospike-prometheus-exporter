FROM golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

ARG VERSION=1.29.0

ADD . $GOPATH/src/github.com/aerospike/aerospike-prometheus-exporter
WORKDIR $GOPATH/src/github.com/aerospike/aerospike-prometheus-exporter
RUN go build -ldflags="-X 'main.version=$VERSION'" -o aerospike-prometheus-exporter ./cmd \
	&& cp aerospike-prometheus-exporter /aerospike-prometheus-exporter

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf

COPY --from=builder /aerospike-prometheus-exporter /usr/bin/aerospike-prometheus-exporter
COPY configs/ape.toml.template /etc/aerospike-prometheus-exporter/ape.toml.template
COPY configs/gauge_stats_list.toml /etc/aerospike-prometheus-exporter/gauge_stats_list.toml
COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN chmod -R a+rwX /etc/aerospike-prometheus-exporter

RUN apk add gettext libintl \
	&& chmod +x /docker-entrypoint.sh

EXPOSE 9145

ENTRYPOINT [ "/docker-entrypoint.sh" ]
CMD ["aerospike-prometheus-exporter", "--config", "/etc/aerospike-prometheus-exporter/ape.toml"]
