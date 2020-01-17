FROM golang:alpine AS builder
ADD . $GOPATH/src/github.com/citrusleaf/aerospike-prometheus-exporter
WORKDIR $GOPATH/src/github.com/citrusleaf/aerospike-prometheus-exporter
RUN apk add git \
  && go get ./... \
  && go build -o aerospike-prometheus-exporter . \
  && cp aerospike-prometheus-exporter /aerospike-prometheus-exporter

FROM golang:alpine
COPY --from=builder /aerospike-prometheus-exporter /usr/bin/aerospike-prometheus-exporter
CMD ["-h localhost", "-p 3000", "-b :9145", "-tags agent,aerospike"]
ENTRYPOINT ["aerospike-prometheus-exporter"]
