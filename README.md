# Aerospike Prometheus Exporter

This repo contains Aerospike's monitoring agent for Prometheus.

This is now in **beta**. If you're an enterprise customer feel free to reach out to support with any questions.
We appreciate feedback from community members on the [issues](https://github.com/aerospike/aerospike-prometheus-exporter/issues).

## Install
1. Install Go v1.12+
1. Run `go get github.com/citrusleaf/aerospike-prometheus-exporter` and cd to it via: `cd $GOPATH/src/github.com/citrusleaf/aerospike-prometheus-exporter`
1. `go build -o aerospike-prometheus-exporter . && ./aerospike-prometheus-exporter -config <full path of the config file>` builds and runs the agent.
    1. for a second agent on the same machine, bind it to a different port.
1. You can generate certificates and set them in the config file in `key_file` and `cert_file` of the `[Agent]` section.
    1. You need to set the `scheme: 'https'` in `scrape_configs:` to be able to ping an agent with TLS enabled.

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
  docker run -itd --name exporter1 -e AS_HOST=172.17.0.2 -e AS_PORT=3000 -e AGENT_TAGS='"agent1","aero_cluster"' aerospike/aerospike-prometheus-exporter:latest
  ```

Enjoy!
