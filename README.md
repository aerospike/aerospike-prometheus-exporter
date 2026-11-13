# Aerospike Prometheus Exporter

This repo contains Aerospike's monitoring agent for Prometheus. The exporter is part of the [Aerospike Monitoring Stack](https://aerospike.com/docs/database/observe/monitor/components).

The Aerospike Prometheus Exporter is now **generally available** (GA).
If you're an enterprise customer feel free to reach out to support with any questions.
We appreciate feedback from community members on the [issues](https://github.com/aerospike/aerospike-prometheus-exporter/issues).

Aerospike agent exports various stats, config from Aerospike Server as metrics in OpenMetrics format.
For more details about Aerospike Server Config refer [Aerospike System Config reference](https://aerospike.com/docs/database/reference/config) and for Aerospike Server Metrics refer [Aerospike System Metrics reference](https://aerospike.com/docs/database/reference/metrics).

**NOTE:** Some of the metrics are pseudo metrics. A pseudo metric is neither a configuration nor a statistic in the server but is exposed as a metric by the agent. For more details, refer to the [Pseudo Metrics](#pseudo-metrics) section.


## Build Instructions

### Aerospike Prometheus Exporter Binary

#### Pre Requisites

- Install Go v1.26.8+

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
     ./aerospike-prometheus-exporter --config <full-path-of-the-config-file> --gauge-list <full-path-of-the-gauge-stats-list-file>
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

####  Pre Requisites for a FIPS build

To generate a FIPS compatible exporter you need Go 1.26.8 or above or have a FIPS enabled OS and OpenSSL.

Aerospike Exporter internally using boringcrypto library for FIPS complaince crypto operations

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

- Build FIPS compliant `deb` package,
    ```bash
    make fips-deb
    ```

- Build FIPS compliant `rpm` package,
    ```bash
    make fips-rpm
    ```

- Build FIPS compliant linux tarball,
    ```bash
    make fips-tar
    ```

Packages will be generated under `./pkg/target/` directory.

#### Support for multiple architectures

Build the exporter packages for `linux/arm64` and `linux/amd64`.

For `arm64`,
```bash
make package-linux-arm64
```

For `amd64`,
```bash
make package-linux-amd64
```

For docker, build and push (for release only) exporter docker image with multiarch support
```bash
make release-docker-multi-arch
```

#### Install Exporter Using `DEB` and `RPM` Packages

- Install `deb` package
    ```bash
    dpkg -i ./pkg/target/aerospike-prometheus-exporter-*.deb
    ```

- Install `rpm` package
    ```bash
    rpm -Uvh ./pkg/target/aerospike-prometheus-exporter-*.rpm
    ```

- Install FIPS compatible `rpm` package
    ```bash
    rpm -Uvh ./pkg/target/aerospike-prometheus-exporter-federal-*.rpm
    ```

- Run the exporter
    ```bash
    systemctl start aerospike-prometheus-exporter.service
    ```

## Aerospike Prometheus Exporter Configuration

- Aerospike Prometheus Exporter requires a configuration file to run. Check [default configuration file](configs/ape.toml).
    ```bash
    mkdir -p /etc/aerospike-prometheus-exporter
    curl https://raw.githubusercontent.com/aerospike/aerospike-prometheus-exporter/master/configs/ape.toml -o /etc/aerospike-prometheus-exporter/ape.toml
    ```

- Edit `/etc/aerospike-prometheus-exporter/ape.toml` to add `db_host` (default `localhost`) and `db_port` (default `3000`) to point to an Aerospike server IP and port.
    ```toml
    [Aerospike]

    db_host="localhost"
    db_port=3000
    ```

- Configure timeout (in seconds) for info commands to Aerospike node (optional). Default value is `5` seconds.
    ```toml
    [Aerospike]
    # timeout for sending commands to the server node in seconds
    timeout=5
    ```

- Update Aerospike security and TLS configurations (optional),
    ```toml
    [Aerospike]

    # TLS certificates.
    # Supports below formats,
    # 2. Certificate file path                                      - "file:<file-path>"
    # 3. Environment variable containing base64 encoded certificate - "env-b64:<environment-variable-that-contains-base64-encoded-certificate>"
    # 4. Base64 encoded certificate                                 - "b64:<base64-encoded-certificate>"
    # Applicable to 'root_ca', 'cert_file' and 'key_file' configurations.

    # root certificate file
    root_ca=""

    # certificate file
    cert_file=""

    # key file
    key_file=""

    # Passphrase for encrypted key_file. Supports below formats,
    # 1. Passphrase directly                                                      - "<passphrase>"
    # 2. Passphrase via file                                                      - "file:<file-that-contains-passphrase>"
    # 3. Passphrase via environment variable                                      - "env:<environment-variable-that-holds-passphrase>"
    # 4. Passphrase via environment variable containing base64 encoded passphrase - "env-b64:<environment-variable-that-contains-base64-encoded-passphrase>"
    # 5. Passphrase in base64 encoded form                                        - "b64:<base64-encoded-passphrase>"
    key_file_passphrase=""

    # node TLS name for authentication
    node_tls_name=""

    # Aerospike cluster security credentials.
    # Supports below formats,
    # 1. Credential directly                                                      - "<credential>"
    # 2. Credential via file                                                      - "file:<file-that-contains-credential>"
    # 3. Credential via environment variable                                      - "env:<environment-variable-that-contains-credential>"
    # 4. Credential via environment variable containing base64 encoded credential - "env-b64:<environment-variable-that-contains-base64-encoded-credential>"
    # 5. Credential in base64 encoded form                                        - "b64:<base64-encoded-credential>"
    # Applicable to 'user' and 'password' configurations.

    # database user
    user=""

    # database password
    password=""

    # authentication mode: internal (server authentication) [default], external (e.g., LDAP), pki. 
    auth_mode=""
    ```

- Update exporter's bind address and port (default: `0.0.0.0:9145`), and add labels.
    ```toml
    [Agent]

    bind=":9145"
    labels={zone="asia-south1-a", platform="google compute engine"}
    ```

- Update exporter's cloud_provider to get few cloud details - region, availability-zone or location.
    ```toml
    [Agent]

    # supported cloud provider values are - AWS, Azure and GCP
    cloud_provider = aws
    ```

- Update exporter's refresh_system_stats to get few system metrics like open-filefd, memory, network recv/trans packets etc.,
    ```toml
    [Agent]

    refresh_system_stats = true
    ```

- Update exporter's OTel configuration to send metrics in OpenTelemetry format
    ```toml
    [Agent]
        enable_open_telemetry = true

    [Agent.OpenTelemetry]
    ```

- Update exporter's OTel grpc_endpoint configuration to send metrics to a gRPC end-point 
    ```toml
    [Agent.OpenTelemetry]
        grpc_endpoint = <endpoint where to push exporter metrics>
    ```

- Update exporter's OTel http_endpoint configuration to send metrics to a HTTP end-point 
    ```toml
    [Agent.OpenTelemetry]
        http_endpoint = <endpoint where to push exporter metrics>
    ```

- Update exporter's OTel headers configuration to send for authentication and additional headers
    ```toml
    [Agent.OpenTelemetry]
        headers = {}
    ```

- Use allowlist and blocklist to filter out required metrics (optional). The allowlist and blocklist supports standard wildcards (globbing patterns which include - `? (question mark)`, `* (asterisk)`, `[ ] (square brackets)`, `{ } (curly brackets)`, `[!]` and `\ (backslash)`) for bulk metrics filtering. For example,
    ```toml
    [Aerospike]

    # Metrics Allowlist - If specified, only these metrics will be scraped. An empty list will include all metrics.
    # Commenting out the below allowlist configs will disable metrics filtering (i.e. all metrics will be scraped).

    # Namespace metrics allowlist
    namespace_metrics_allowlist=[
    "client_read_[a-z]*",
    "stop_writes",
    "storage-engine.file*",
    "storage-engine.file\\[*\\].*",
    "storage-engine.file*defrag_q",
    "client_write_success",
    "*memory-pct",
    "objects",
    "*_available_pct"
    ]

    # Set metrics allowlist, An empty list will include all metrics.
    # below example, only 2 metrics mentioned will be scraped
    set_metrics_allowlist=[
    "objects",
    "tombstones"
    ]

    # Node metrics allowlist, An empty list will include all metrics.
    node_metrics_allowlist=[
    "uptime",
    "cluster_size",
    "batch_index_*",
    "xdr_ship_*"
    ]

    # XDR metrics allowlist (only for Aerospike versions 5.0 and above)
    xdr_metrics_allowlist=[
    "success",
    "latency_ms",
    "throughput",
    "lap_us"
    ]

    # Secondary index metrics allowlist, An empty list will include all metrics.
    sindex_metrics_allowlist = [
    "entries",
    "ibtr_memory_used",
    "nbtr_memory_used",
    "query_basic_complete",
    "query_basic_error",
    "query_basic_abort",
    "query_basic_avg_rec_count"
    ]

    # Metrics Blocklist - If specified, these metrics will be NOT be scraped.

    # Namespace metrics blocklist, An empty list means all metrics are allowed.
    # to exclude or stop all metrics, use "*"
    namespace_metrics_blocklist=[
    "memory_used_sindex_bytes",
    "client_read_success"
    ]

    # Set metrics blocklist, blocks all set metrics, An empty list means all metrics are allowed.
    # to exclude or stop all metrics, use ["*"]
    set_metrics_blocklist=["*"]

    # Node metrics blocklist
    node_metrics_blocklist=[
    "batch_index_*_buffers"
    ]

    # XDR metrics blocklist (only for Aerospike versions 5.0 and above), An empty list means all metrics are allowed.
    # to exclude or stop all metrics, use ["*"]
    # xdr_metrics_blocklist=[]

    # Secondary index metrics blocklist, An empty list means all metrics are allowed.
    # to exclude or stop all metrics, use ["*"]
    # sindex_metrics_blocklist = []
    ```

- To enable basic HTTP authentication and/or enable HTTPS between the Prometheus server and the exporter, use the below configurations keys,
    ```toml
    [Agent]
    # Exporter HTTPS (TLS) configuration
    # HTTPS between Prometheus and Exporter

    # TLS certificates.
    # Supports below formats,
    # 1. Certificate file path                                      - "file:<file-path>"
    # 2. Environment variable containing base64 encoded certificate - "env-b64:<environment-variable-that-contains-base64-encoded-certificate>"
    # 3. Base64 encoded certificate                                 - "b64:<base64-encoded-certificate>"
    # Applicable to 'root_ca', 'cert_file' and 'key_file' configurations.

    # Server certificate
    cert_file = ""

    # Private key associated with server certificate
    key_file = ""

    # Root CA to validate client certificates (for mutual TLS)
    root_ca = ""

    # Golang - refer documentation https://pkg.go.dev/crypto/tls#pkg-constants of golang CipherSuites for TLS >=1.2 (both supported and Insecure)
    # a comma separated TLS Cipher suites to use, example: TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384
    # NOTE: Cipher configuration is support only till TLS1.2 verison and not possible in TLS1.3
    tls_cipher_suites = ""

    # Passphrase for encrypted key_file. Supports below formats,
    # 1. Passphrase directly                                                      - "<passphrase>"
    # 2. Passphrase via file                                                      - "file:<file-that-contains-passphrase>"
    # 3. Passphrase via environment variable                                      - "env:<environment-variable-that-holds-passphrase>"
    # 4. Passphrase via environment variable containing base64 encoded passphrase - "env-b64:<environment-variable-that-contains-base64-encoded-passphrase>"
    # 5. Passphrase in base64 encoded form                                        - "b64:<base64-encoded-passphrase>"
    key_file_passphrase = ""

    # Basic HTTP authentication for '/metrics'.
    # Supports below formats,
    # 1. Credential directly                                                      - "<credential>"
    # 2. Credential via file                                                      - "file:<file-that-contains-credential>"
    # 3. Credential via environment variable                                      - "env:<environment-variable-that-contains-credential>"
    # 4. Credential via environment variable containing base64 encoded credential - "env-b64:<environment-variable-that-contains-base64-encoded-credential>"
    # 5. Credential in base64 encoded form                                        - "b64:<base64-encoded-credential>"
    basic_auth_username=""
    basic_auth_password=""
    ```
- NOTE: Minimum TLS version is 1.2, tls_cipher_suites can be configured only upto TLS1.2
  
- Use users' allowlist and blocklist configuration to filter out the users for which the statistics are to be fetched. The user statistics are available in Aerospike 5.6+. To fetch user statistics, the authenticated user must have `user-admin` privilege.
    ```toml
    [Aerospike]

    # Users Statistics (user statistics are available in Aerospike 5.6+)
    # Users Allowlist and Blocklist to control for which users their statistics should be collected.
    # Note globbing patterns are not supported for this configuration.

    user_metrics_users_allowlist=[
    "admin",
    "superuser",
    "aerospikeUser1",
    "aerospikeUser2"
    ]

    user_metrics_users_blocklist=[
    "admin",
    "superuser"
    ]
    ```

- Exporter logs to console by default. To enable file logging, use `log_file` configuration to specify a file path. Use `log_level` configuration to specify a logging level (optional). Default logging level is `info`.
    ```toml
    [Agent]
    # Exporter logging configuration
    # Log file path (optional, logs to console by default)
    # Level can be info|warning,warn|error,err|debug|trace ('info' by default)
    log_file = ""
    log_level = ""
    ```

- Use `latency_buckets_count` to specify number of histogram buckets to be exported for latency metrics (optional). Bucket thresholds range from 2<sup>0</sup> to 2<sup>16</sup> (`17` buckets). All threshold buckets are exported by default (`latency_buckets_count=0`).

    Example, `latency_buckets_count=5` will export first five buckets i.e. `<=1ms`, `<=2ms`, `<=4ms`, `<=8ms` and `<=16ms`.
    ```toml
    [Aerospike]
    # Number of histogram buckets to export for latency metrics. Bucket thresholds range from 2^0 to 2^16 (17 buckets).
    # e.g. latency_buckets_count=5 will export first five buckets i.e. <=1ms, <=2ms, <=4ms, <=8ms and <=16ms.
    # Default: 0 (export all threshold buckets).
    latency_buckets_count=0
    ```

## Secret Agent

Secret Agent support is optional. An existing `ape.toml` keeps working when `[Agent.SecretAgent]` is absent.

APE uses the same Secret Agent transports as Aerospike Server: plain TCP, TCP with TLS, TCP with mutual TLS, and a Unix domain socket. HTTP and HTTPS are not used. Set either `socket_path` or `address` and `port`. TLS is selected when `tls_name` and `ca_file` are set, and mutual TLS also requires `cert_file` and `key_file`. Certificate verification stays enabled. The certificates used to connect to Secret Agent cannot themselves be `secrets:` references.

Sensitive configuration values can use a Secret Agent reference. APE fetches that value during startup, before it initializes the Aerospike connection, the Prometheus listener, or OpenTelemetry. `secrets:<resource>:<key>` uses `SecretValue` as text, including a PEM certificate. `secrets:b64:<resource>:<key>` base64-decodes `SecretValue` once. Use the base64 form when the stored value is base64, or when Secret Agent `convert-to-base64` is enabled. Existing direct text, `file:`, `env:`, `env-b64:`, and `b64:` values continue to work.

```toml
[Agent.SecretAgent]
address = "127.0.0.1"
port = 3005
timeout = 5

[Aerospike]
user = "secrets:APE:DbUser"
password = "secrets:APE:DbPassword"
cert_file = "secrets:b64:APE:Cert"
```

`make test` runs the in-process unit tests and does not need Docker. `make test-integration` runs every Docker suite that can run on a laptop: Secret Agent, then OpenTelemetry, then the processor info suite, then the user-stats suite, then a Prometheus scrape of Community 7.2. `make test-integration-aws` stays separate because it needs AWS credentials and a Secrets Manager secret.

`make test-integration-processor` starts one Enterprise server from `ci/processor/aerospike.conf`, runs the namespace and node info commands, and passes those strings to the processors. It checks that every numeric field is emitted, including storage-engine and flash index array stats. The test also creates secondary index `name_idx`, writes and reads records, and checks the set, sindex, and latency processors against those responses. Set `APE_IT_FEATURE_KEY_FILE` to an Enterprise feature key on the host. The key is mounted into the container and is not committed. When the variable is unset the command prints a skip message and exits 0.

`make test-integration-users` starts a separate Enterprise server with `security` and `enable-quotas` enabled. The server's default `admin` / `admin` user creates `u13user`, `phani`, and `u1phani`. `asbench` then runs as each of those users, and the user processor is checked against `QueryUsers` while that load is running. Connections and single-record read and write rates must be above zero. The roles have no rate quota, so the quota stats are present and zero. Set `APE_IT_FEATURE_KEY_FILE` to an Enterprise feature key on the host. When the variable is unset the command prints a skip message and exits 0.

`make test-integration-versions` starts a Community server with the image's own configuration, runs the exporter, and waits until `/metrics` shows `aerospike_node_up` 1 plus a namespace stat and a node stat. Server 5.7 uses the image's file-backed device namespace, because that image rejects `file` inside `storage-engine memory`. The default tag is `7.2.0.11`. Set `APE_IT_SERVER_VERSIONS` to a space-separated list of tags to scrape more than one release. The Integrations server versions workflow scrapes 5.7, 6.4, 7.2, 8.0, 8.1, and 8.2 on pull requests to master and dev, and nightly at 03:00 UTC. The Integration workflow scrapes 7.2 on branch pushes.

`make test-integration-secret-agent` starts Secret Agent with its file backend. Two exporters are started from `ape.toml` files: `ci/secret-agent/ape.toml` connects to Secret Agent over TCP, and `ci/secret-agent/ape-uds.toml` connects over a Unix socket. That suite generates TLS material at runtime and does not commit certificates or private keys. The same runtime material is stored for Secret Agent as text and as base64, then fetched with `secrets:` and `secrets:b64:` and used for a mutual-TLS handshake.

`make test-integration-otel` runs the OpenTelemetry exporter against an OpenTelemetry collector. Compose starts a single-node Aerospike Community Edition server, the collector, and the exporter. The exporter sends OTLP over HTTP to `http://otel-collector:4318/v1/metrics`. The collector writes metrics to a file, and the test waits until `aerospike.server.node_up` and `aerospike.server.node_stats.cluster_size` are present. The gRPC client uses TLS, so this suite uses `http_endpoint`.

### AWS Secrets Manager TLS/mTLS suite

`make test-integration-aws` runs the exporter's HTTPS listener end to end with certificates that come from AWS Secrets Manager through Secret Agent. Each run generates a fresh CA, server certificate, encrypted and unencrypted server keys, and client certificates. It writes them to the secret, starts Secret Agent and the exporter, runs the checks, and then removes the containers and the generated files. The secret value is replaced with a placeholder at the end of the run. When `APE_IT_AWS_SECRET_ARN` is unset the command prints a skip message and exits 0, so CI can run it before that secret exists. `make test-integration` does not read this variable.

The secret must be disposable test data. The suite refuses to write to it unless it carries the tag `ape-integration-test=true`:

```sh
aws secretsmanager tag-resource --secret-id "$ARN" --tags Key=ape-integration-test,Value=true
APE_IT_AWS_SECRET_ARN="$ARN" make test-integration-aws
```

AWS credentials come from the default chain. `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, and `AWS_PROFILE` are passed through, and `~/.aws` is mounted read-only. With SSO, export temporary credentials first, for example `eval "$(aws configure export-credentials --format env)"`. The identity needs `secretsmanager:DescribeSecret`, `secretsmanager:PutSecretValue`, and `secretsmanager:GetSecretValue` on that secret.

| Variable | Default | Purpose |
|---|---|---|
| `APE_IT_AWS_SECRET_ARN` | unset skips the suite | Secret holding `cert_file`, `key_file`, `key_file_plain`, `root_ca`, `key_file_passphrase`, and `key_file_passphrase_invalid` |
| `APE_IT_AWS_REGION` | region in the ARN | Region for Secret Agent and the AWS CLI |
| `APE_IT_AWS_MISSING_SECRET_ARN` | a non-existent ARN in the same account | Used for the missing-secret check |
| `APE_IT_SA_TRANSPORT` | `tcp` | `tcp`, `tls`, `mtls`, or `uds` between the exporter and Secret Agent |
| `APE_IT_PEM_ENCODING` | `base64` | `base64` stores PEM values base64-encoded and reads them with `secrets:b64:`. `text` stores PEM as text and reads it with `secrets:` |
| `APE_IT_SA_BACKEND` | `aws` | `file` runs the same checks with Secret Agent's file backend and no AWS access |
| `APE_IT_AWS_PLAIN_SECRET_ARN` | unset skips that one test | Secret whose `username` and `password` keys are plain text. A second Secret Agent has `convert-to-base64` on and the exporter reads them with `secrets:b64:APEPlain:username` and `secrets:b64:APEPlain:password` |
| `APE_IT_PLAIN_USERNAME` | required with the plain ARN | Expected `username` value |
| `APE_IT_PLAIN_PASSWORD` | required with the plain ARN | Expected `password` value |

The default is base64 because Aerospike's Secret Agent guidance is to store multi-line values such as certificates base64-encoded in AWS Secrets Manager. Passphrases are single-line and are read with `secrets:`.

The suite checks HTTPS with a trusted CA and an untrusted CA, mutual TLS with a valid client certificate, no client certificate, and a certificate from another CA, an encrypted server key unlocked by a passphrase from the secret, and startup failure for a wrong passphrase, an unknown alias, a missing key, and a missing AWS secret. It also checks that exporter output never contains a passphrase or private-key material.

A separate secret covers plain text that Secret Agent base64-encodes. Create it with these keys, leave the values as text, and tag it `ape-integration-test=true`:

```json
{"username": "<plain username>", "password": "<plain password>"}
```

The suite does not write this secret. Set `APE_IT_AWS_PLAIN_SECRET_ARN`, `APE_IT_PLAIN_USERNAME`, and `APE_IT_PLAIN_PASSWORD`. The exporter reads `secrets:b64:APEPlain:username` and `secrets:b64:APEPlain:password`. Without the ARN, that one test is skipped.

## Pseudo Metrics

| S.No | Name  | Description  |
|------|-------|-------------|
| 1    | aerospike_node_up | This metric is returned by the exporter when it is up and running, regardless of whether it can connect to the Aerospike Server. |
| 2    | aerospike_node_stats_pseudo_log_debug | Indicates whether DEBUG logging is enabled for a node in an Aerospike Cluster. `1` means ENABLED, `0` means DISABLED. |
| 3    | aerospike_node_stats_pseudo_log_detail | Indicates whether DETAIL logging is enabled for a node in an Aerospike Cluster. `1` means ENABLED, `0` means DISABLED. |
