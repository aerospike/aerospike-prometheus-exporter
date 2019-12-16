This repo contains Aerospike's monitoring solution with Prometheus + Grafana.

Steps to use:

1. Install Go v1.12+, and Docker for your platform.
2. Run `go get github.com/citrusleaf/aerospike-monitoring` and cd to it via: `cd $GOPATH/src/github.com/citrusleaf/aerospike-monitoring`
3. `go build -o aerospike-prometheus-exporter . && ./aerospike-prometheus-exporter -h <server_node> -p 3000 -b :9145 -tags agent1,very_nice` builds and runs the agent.
3.1. for a second agent on the same machine, bind it to a different port: `go build . && ./aerospike-prometheus-exporter -h <server_node> -p 3000 -b :9146 -tags agent1,very_nice`
4. Edit `docker/prometheus/prometheus.yml` and change the target IPs to aerospike-prometheus-exporter installations.
5. Run `docker-compose up` to download, build and run the docker images. To stop the containers, run `docker-compose down`
6. Go to your browser and use the URL: `http://localhost:3000`. User/Pass is `admin/pass`
7. Prometheus dashboard is at: `http://localhost:9090`
7.1. To make a dashboard your default, first choose that dashboard, and then star it on the toolbar on top right of the screen. After that, you can go to grafana preferences and choose that starred dashboard as default.

Enjoy!

Please send your feedback via email to khosrow@aerospike.com
