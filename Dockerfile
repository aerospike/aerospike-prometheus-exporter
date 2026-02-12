FROM golang:1.25-alpine@sha256:f6751d823c26342f9506c03797d2527668d095b0a15f1862cddb4d927a7a4ced AS builder

ARG VERSION=1.29.0

ADD . $GOPATH/src/github.com/aerospike/aerospike-prometheus-exporter
WORKDIR $GOPATH/src/github.com/aerospike/aerospike-prometheus-exporter
RUN go build -ldflags="-X 'main.version=$VERSION'" -o aerospike-prometheus-exporter ./cmd \
	&& cp aerospike-prometheus-exporter /aerospike-prometheus-exporter

FROM alpine:3.23.3@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

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
