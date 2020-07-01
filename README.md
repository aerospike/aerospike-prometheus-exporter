# Aerospike Prometheus Exporter

This repo contains Aerospike's monitoring agent for Prometheus. The exporter is part of the [Aerospike Monitoring Stack](https://www.aerospike.com/docs/tools/monitorstack/index.html).

The Aerospike Prometheus Exporter is now **generally available** (GA).
If you're an enterprise customer feel free to reach out to support with any questions.
We appreciate feedback from community members on the [issues](https://github.com/aerospike/aerospike-prometheus-exporter/issues).

## Build Instructions

### Aerospike Prometheus Exporter Binary

#### Pre Requisites

- Install Go v1.12+

#### Steps

1. Clone or fetch this repository
    ```bash
    git clone https://github.com/aerospike/aerospike-prometheus-exporter.git
    cd aerospike-prometheus-exporter/

    # or

    # go get github.com/aerospike/aerospike-prometheus-exporter
    # cd $GOPATH/src/github.com/aerospike/aerospike-prometheus-exporter
    ```
2. Build the exporter binary,
    ```bash
    go build -o aerospike-prometheus-exporter .
    ```
    or,

    ```bash
    make
    ```

3. Run the exporter
     ```bash
     ./aerospike-prometheus-exporter --config <full-path-of-the-config-file>
     ```

### Aerospike Prometheus Exporter Docker Image

- Build the docker image
  ```bash
  docker build . -t aerospike/aerospike-prometheus-exporter:latest
  ```
  or,

  ```bash
  make docker
  ```
- Run the exporter as a container
  ```bash
  docker run -itd --name exporter1 -e AS_HOST=172.17.0.2 -e AS_PORT=3000 -e METRIC_LABELS="type='development',source='aerospike'" aerospike/aerospike-prometheus-exporter:latest
  ```

### `RPM`, `DEB` and `tar` Package

####  Pre Requisites

- FPM Package manager
    - https://fpm.readthedocs.io/en/latest/installing.html
    - For instance, on Debian-derived systems (Debian, Ubuntu, etc),
        ```bash
        apt install ruby ruby-dev rubygems build-essential -y
        ```
        ```bash
        gem install --no-document fpm
        ```

#### Steps

Build the exporter go binary and package it into `rpm`, `deb` or `tar`.

- Build `deb` package,
    ```bash
    make deb
    ```

- Build `rpm` package,
    ```bash
    make rpm
    ```

- Build linux tarball,
    ```bash
    make tar
    ```

Packages will be generated under `./pkg/target/` directory.

#### Install Exporter Using `DEB` and `RPM` Packages

- Install `deb` package
    ```bash
    dpkg -i ./pkg/target/aerospike-prometheus-exporter-*.deb
    ```

- Install `rpm` package
    ```bash
    rpm -Uvh ./pkg/target/aerospike-prometheus-exporter-*.rpm
    ```

- Run the exporter
    ```bash
    systemctl start aerospike-prometheus-exporter.service
    ```

## Aerospike Prometheus Exporter Configuration

- Aerospike Prometheus Exporter requires a configuration file to run. Check [default configuration file](ape.toml).
    ```bash
    mkdir -p /etc/aerospike-prometheus-exporter
    curl https://raw.githubusercontent.com/aerospike/aerospike-prometheus-exporter/master/ape.toml -o /etc/aerospike-prometheus-exporter/ape.toml
    ```

- Edit `/etc/aerospike-prometheus-exporter/ape.toml` to add `db_host` (default `localhost`) and `db_port` (default `3000`) to point to an Aerospike server IP and port.
    ```toml
    [Aerospike]

    db_host="localhost"
    db_port=3000
    ```
- Update Aerospike security and TLS configurations (optional),
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

- Use metrics allowlist to filter out required metrics (optional). The allowlist supports standard wildcards (globbing patterns which include - `? (question mark)`, `* (asterisk)`, `[ ] (square brackets)`, `{ } (curly brackets)`, `[!]` and `\ (backslash)`) for bulk allowlisting. For example,
    ```toml
    [Aerospike]

    # Metrics Allowlist - If specified, only these metrics will be scraped. An empty list will exclude all metrics.
    # Commenting out the below allowlist configs will disable metrics filtering (i.e. all metrics will be scraped).

    # Namespace metrics allowlist
    namespace_metrics_allowlist=[
    "client_read_[a-z]*",
    "stop_writes",
    "storage-engine.file.defrag_q",
    "client_write_success",
    "memory_*_bytes",
    "objects",
    "*_available_pct"
    ]

    # Set metrics allowlist
    set_metrics_allowlist=[
    "objects",
    "tombstones"
    ]

    # Node metrics allowlist
    node_metrics_allowlist=[
    "uptime",
    "cluster_size",
    "batch_index_*",
    "xdr_ship_*"
    ]

    # XDR metrics allowlist (only for server versions 5.0 and above)
    xdr_metrics_allowlist=[
    "success",
    "latency_ms",
    "throughput",
    "lap_us"
    ]
    ```

- To enable basic HTTP authentication and/or enable HTTPS between the Prometheus server and the exporter, use the below configurations keys,

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