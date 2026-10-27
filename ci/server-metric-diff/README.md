# Server metric compatibility audit

Status: deferred. This is a saved test strategy, not part of the current
feature work. The implementation was removed so it does not ride along with
the feature branch. Do not add it to pull-request CI until it is picked up
again.

When revisited, run the current exporter against an older and a newer
Aerospike Enterprise server. Record the exact info commands and responses the
exporter requests, then compare numeric and boolean stat names with the
Prometheus families from that same scrape.

- `raw_only` is a stat the server sent that the exporter did not emit. That is
  the signal for a silently dropped metric.
- `non_metric` covers metadata, labels, strings, and structured fields.
- The first runs stay report-only. Do not fail on `raw_only` until legitimate
  exceptions are classified.
