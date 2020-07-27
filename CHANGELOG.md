# Changelog

This file documents all notable changes to Aerospike Prometheus Exporter

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