# Aerospike Prometheus Exporter

This repo contains Aerospike's monitoring agent for Prometheus.

This is now in **beta**. If you're an enterprise customer feel free to reach out to support with any questions.
We appreciate feedback from community members on the [issues](https://github.com/aerospike/aerospike-prometheus-exporter/issues).

## Install
1. Install Go v1.12+
2. Run `go get github.com/aerospike/aerospike-prometheus-exporter` and cd to it via: `cd $GOPATH/src/github.com/aerospike/aerospike-prometheus-exporter`
3. `go build -o aerospike-prometheus-exporter . && ./aerospike-prometheus-exporter -config <full path of the config file>` builds and runs the agent.
    - for a second agent on the same machine, bind it to a different port.
4. You can generate certificates and set them in the config file in `key_file` and `cert_file` of the `[Agent]` section.
    - You need to set the `scheme: 'https'` in `scrape_configs:` to be able to ping an agent with TLS enabled.

## Build Docker Image

- Clone the repo
  ```
  git clone https://github.com/aerospike/aerospike-prometheus-exporter.git
  cd aerospike-prometheus-exporter
  ```
- Build the docker image
  ```
  docker build . -t aerospike/aerospike-prometheus-exporter:latest
  ```
- Example run
  ```
  docker run -itd --name exporter1 -e AS_HOST=172.17.0.2 -e AS_PORT=3000 -e METRIC_LABELS="type='development',source='aerospike'" aerospike/aerospike-prometheus-exporter:latest
  ```

## Aerospike Prometheus Exporter Configuration

- Aerospike Prometheus Exporter requires a configuration file to run. Check [sample configuration file](aeroprom.conf.dev).
    ```
    mkdir -p /etc/aerospike-prometheus-exporter
    curl https://raw.githubusercontent.com/aerospike/aerospike-prometheus-exporter/master/aeroprom.conf.dev -o /etc/aerospike-prometheus-exporter/ape.toml
    ```

- As a minimum required configuration, edit `/etc/aerospike-prometheus-exporter/ape.toml` to add `db_host` and `db_port` to point to an Aerospike server IP and port.
    ```toml
    [Aerospike]

    db_host="localhost"
    db_port=3000
    ```
- Update Aerospike security and TLS configurations (if applicable),
    ```toml
    [Aerospike]

    # certificate file
    cert_file=""

    # key file
    key_file=""

    # Passphrase for encrypted key_file. Supports below formats,
    # 1. Passphrase directly                 - "<passphrase>"
    # 2. Passphrase via file                 - "file:<file-that-contains-passphrase>"
    # 3. Passphrase via environment variable - "env:<environment-variable-that-holds-passphrase>"
    key_file_passphrase=""

    # node TLS name for authentication
    node_tls_name=""

    # root certificate file
    root_ca=""

    # authentication mode: internal (for server), external (LDAP, etc.)
    auth_mode=""

    # database user
    user=""

    # database password
    password=""
    ```

- Update exporter's bind address and port (default: `0.0.0.0:9145`), and add labels.
    ```toml
    [Agent]

    bind=":9145"
    labels={zone="asia-south1-a", platform="google compute engine"}
    ```

- Use metrics whitelist to filter out required metrics (optional). The whitelist supports standard wildcards (globbing patterns which include - `? (question mark)`, `* (asterisk)`, `[ ] (square brackets)`, `{ } (curly brackets)`, `[!]` and `\ (backslash)`) for bulk whitelisting. For example,
    ```toml
    [Aerospike]

    # Metrics Whitelist - If specified, only these metrics will be scraped. An empty list will exclude all metrics.
    # Commenting out the below whitelist configs will disable whitelisting (all metrics will be scraped).

    # Namespace metrics whitelist
    namespace_metrics_whitelist=[
    "client_read_[a-z]*",
    "stop_writes",
    "storage-engine.file.defrag_q",
    "client_write_success",
    "memory_*_bytes",
    "objects",
    "*_available_pct"
    ]

    # Set metrics whitelist
    set_metrics_whitelist=[
    "objects",
    "tombstones"
    ]

    # Node metrics whitelist
    node_metrics_whitelist=[
    "uptime",
    "cluster_size",
    "batch_index_*",
    "xdr_ship_*"
    ]

    # XDR metrics whitelist (only for server versions 5.0 and above)
    xdr_metrics_whitelist=[
    "success",
    "latency_ms",
    "throughput",
    "lap_us"
    ]
    ```

- To enable basic HTTP authentication and/or enable HTTPS between the Prometheus server and the exporter, use the below configurations keys (optional),

  ```toml
  [Agent]

  # File paths should be double quoted.

  # Certificate file for the metric servers for prometheus
  cert_file = ""

  # Key file for the metric servers for prometheus
  key_file = ""

  # Basic HTTP authentication for '/metrics'.
  basic_auth_username=""
  basic_auth_password=""
  ```
