# Changelog

This file documents all notable changes to Aerospike Prometheus Exporter


## [v1.3.0](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.3.0)

### Features
- [PROD-1742] - Added support for user statistics
    - Per-user statistics are available in Aerospike 5.6+.


## [v1.2.1](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.2.1)

### Improvements
- Use Aerospike Go client v5 to support Aerospike server 5.6.
- [PROD-1774] - Add new connections opened/closed statistics introduced in 5.6
    - `client_connections_opened`
    - `client_connections_closed`
    - `heartbeat_connections_opened`
    - `heartbeat_connections_closed`
    - `fabric_connections_opened`
    - `fabric_connections_closed`
- [PROD-1775] - Add new all flash statistics introduced in 5.6
    - `index_flash_alloc_bytes`
    - `index_flash_alloc_pct`
- Add other new metrics introduced in 5.6,
    - `memory_used_set_index_bytes`
    - `fail_client_lost_conflict`
    - `fail_xdr_lost_conflict`
    - `threads_joinable`
    - `threads_detached`
    - `threads_pool_total`
    - `threads_pool_active`


## [v1.2.0](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.2.0)

### Features
- [TOOLS-1686] - Added support for mutual authentication and encrypted key file for HTTPS between Prometheus and Exporter.

### Improvements
- [PROD-1660] - When starting under systemd, wait for network to be online.
- Add new metrics,
  - `dup_res_ask`
  - `dup_res_respond_read`
  - `dup_res_respond_no_read`
  - `xdr_bin_cemeteries`
  - `conflict-resolve-writes`

### Other Changes
- Aerospike Prometheus Exporter is now built with Go 1.16.
    - The deprecated, legacy behavior of treating the CommonName field on X.509 serving certificates as a host name when no Subject Alternative Names are present is now disabled by default. It can be temporarily re-enabled by adding the value `x509ignoreCN=0` to the `GODEBUG` environment variable.


## [v1.1.6](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.1.6)

### Improvements
- [TOOLS-1614] - Fetch credentials and certificates via file, environment variables and in base64 encoded form.
- Added new XDR (per DC) metric - `nodes`.


## [v1.1.5](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.1.5)

### Improvements
- [TOOLS-1603] - Use latency info commands based on Aerospike build version.
- [TOOLS-1602] - Add device or file name as a label in `storage-engine` statistics.

### Fixes
- [TOOLS-1595] - Fix info commands to get address and port for TLS and non-TLS service
- [TOOLS-1601] - Add constant labels to `aerospike_node_up` metric


## [v1.1.4](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.1.4)

### Improvements
- Add new metrics and configs introduced in Aerospike 5.2
    - `device_data_bytes`
    - `xdr-bin-tombstone-ttl`
    - `ignore-migrate-fill-delay`
- Add `--version` command line option


## [v1.1.3](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.1.3)

### Improvements
- Add `/health` and `/` url endpoints
- Add some missing namespace, node and xdr statistics

### Fixes
- Fix concurrent map writes
- Add mutex to protect against concurrent collection


## [v1.1.2](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.1.2)

### Improvements
- Optimize latency data export - use only non-zero buckets.
- Use `latencies` info command against Aerospike versions 5.1 and above.

### Fixes
- Retry for connection, network or timeout errors.


## [v1.1.1](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.1.1)

### Fixes
- Tolerate older configurations - metrics' `_whitelist` and `_blacklist`


## [v1.1.0](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.1.0)

### Features
- Add support for using a `blocklist` to filter metrics.
- Export latency metrics as Prometheus histograms.

### Improvements
- Remove accumulated metric `aerospike_node_info` to prevent cardinality explosion.
- Reuse connections to aerospike node.
- Package Aerospike Prometheus Exporter as `.rpm`/`.deb`/`.tar`.
- Add `trace` level logging.

### Fixes
- Fix glob pattern regex used for metric allowlist and blocklist.

### Other Changes
- Rename configurations `_whitelist` to `_allowlist`.
- Rename `aeroprom.conf.dev` to `ape.toml`.
- Update docker image entrypoint script and config files,
    - Add `log_file` config.
    - Remove `update_interval` unused config.
    - Fix `password` config.
- Cleanup unused configurations


## [v1.0.0](https://github.com/aerospike/aerospike-prometheus-exporter/releases/tag/v1.0.0)

- Initial Release.