FROM golang:1.16-alpine AS builder

ARG TARGETARCH
ARG TARGETOS
ARG VERSION=v1.9.0

ADD . $GOPATH/src/github.com/aerospike/aerospike-prometheus-exporter
WORKDIR $GOPATH/src/github.com/aerospike/aerospike-prometheus-exporter
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-X 'main.version=$VERSION'" -o aerospike-prometheus-exporter . \
	&& cp aerospike-prometheus-exporter /aerospike-prometheus-exporter

FROM alpine:latest

COPY --from=builder /aerospike-prometheus-exporter /usr/bin/aerospike-prometheus-exporter
COPY ape.toml.template /etc/aerospike-prometheus-exporter/ape.toml.template
COPY docker-entrypoint.sh /docker-entrypoint.sh

RUN apk add gettext libintl \
	&& chmod +x /docker-entrypoint.sh

EXPOSE 9145

ENTRYPOINT [ "/docker-entrypoint.sh" ]
CMD ["aerospike-prometheus-exporter", "--config", "/etc/aerospike-prometheus-exporter/ape.toml"]
