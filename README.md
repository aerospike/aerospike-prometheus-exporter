# Aerospike Prometheus Exporter

This repo contains Aerospike's monitoring agent for Prometheus.

## Steps to use:

1. Install Go v1.12+
1. Run `go get github.com/citrusleaf/aerospike-prometheus-exporter` and cd to it via: `cd $GOPATH/src/github.com/citrusleaf/aerospike-prometheus-exporter`
1. `go build -o aerospike-prometheus-exporter . && ./aerospike-prometheus-exporter -h <server_node> -p 3000 -b :9145 -tags agent1,very_nice` builds and runs the agent.
    1. for a second agent on the same machine, bind it to a different port: `go build . && ./aerospike-prometheus-exporter -h <server_node> -p 3000 -b :9146 -tags agent1,very_nice`

## Build Docker Image

- Clone the repo
  ```
  git clone https://github.com/citrusleaf/aerospike-prometheus-exporter.git
  cd aerospike-prometheus-exporter
  ```
- Build the docker image
  ```
  docker build . -t aerospike/aerospike-prometheus-exporter:latest
  ```
- Example run
  ```
  docker run -itd --name exporter1  aerospike/aerospike-prometheus-exporter:latest -h 172.17.0.2 -p 3000 -b :9145 -tags agent1,aero_cluster
  ```

Enjoy!
