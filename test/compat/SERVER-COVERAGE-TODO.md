# Server metric coverage audit (outstanding)

The cross-version half of the original strategy is now implemented in
[compat-test.sh](compat-test.sh): it runs an old and a new exporter against
three Aerospike server versions and fails if a newer combination loses a
metric family, decreases the gauge count, reverts a gauge to a counter, or
empties a whole context.

What remains unimplemented is the **server coverage** check, which answers a
different question. `compat-test.sh` compares exporter output to exporter
output, so it can only prove the exporter did not regress. It cannot reveal
that a new server version ships stats nobody has wired up yet — those are
invisible on both sides of every comparison.

## What to build

For one scrape, record the exact info commands the exporter issues and the
responses it gets back, then compare the numeric and boolean stat names in
those responses against the Prometheus families from that same scrape.

- `raw_only` is a stat the server sent that the exporter did not emit. That is
  the signal for a silently dropped or never-implemented metric.
- `non_metric` covers metadata, labels, strings, and structured fields, which
  are legitimately absent from the output.

Keep it report-only. Do not fail on `raw_only` until the legitimate exceptions
are classified, because the first runs will be dominated by them.

## Known gaps in compat-test.sh itself

These are coverage holes in the implemented harness, not part of the audit
above:

- No traffic is generated, so 8 `aerospike_latencies_*` families (write, read
  and batch `_bucket` / `_count`) never appear and are not checked.
- `users` and `sysinfo` report zero families: security is not configured and
  `refresh_system_stats` is off.
- `checkpoint_*` metrics need a server started with `--preview=index-checkpoint`
  and an `index-checkpoint-path`, which no config profile provides.
- The exporter image-identity check compares only the first server's two cells.
- The released baseline in [docker-compose.yml](docker-compose.yml)
  (`OLD_EXPORTER_IMAGE`) must be bumped by hand on each release.
