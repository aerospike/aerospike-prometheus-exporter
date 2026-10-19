#!/usr/bin/env bash
#
# Cross-version metric compatibility check. No setup required.
#
# Builds the exporter from this working tree, runs it and the last released
# exporter against three Aerospike server versions, and reports whether the
# newer exporter or newer server lost any metrics.
#
#   ./compat-test.sh
#   ./compat-test.sh --server-rpm <path.rpm>     test an unreleased server
#   ./compat-test.sh --new-exporter <image>      compare a released image instead
#   ./compat-test.sh --old-exporter <image>      change the released baseline
#   ./compat-test.sh --community                 force Community Edition
#   ./compat-test.sh --keep                      leave containers up
#
# An Aerospike Enterprise feature key is found automatically in ~/.aerospike/,
# ./, ~/Downloads/ or /etc/aerospike/, or via FEATURE_KEY=<path>. With no key
# it falls back to Community Edition, which cannot cover the xdr context.
#
# Exits non-zero on regression.

set -uo pipefail
cd "$(dirname "$0")" || exit 1
REPO_ROOT=$(cd ../.. && pwd)

KEEP=0
SERVER_RPM=""
COMMUNITY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --keep)          KEEP=1; shift ;;
    --community)     COMMUNITY=1; shift ;;
    --server-rpm)    SERVER_RPM=${2:?path required}; shift 2 ;;
    --new-exporter)  export NEW_EXPORTER_IMAGE=${2:?image required}; shift 2 ;;
    --old-exporter)  export OLD_EXPORTER_IMAGE=${2:?image required}; shift 2 ;;
    -h|--help)       sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *)               echo "unknown argument: $1"; exit 1 ;;
  esac
done

SERVERS="a b c"
EXPORTERS="old new"
WORK=$(mktemp -d)
FAILED=0

say()  { printf '%s\n' "$*"; }
pass() { printf '  \033[32mPASS\033[0m  %s\n' "$*"; }
fail() { printf '  \033[31mFAIL\033[0m  %s\n' "$*"; FAILED=1; }
info() { printf '  ----  %s\n' "$*"; }
compose() { docker compose "$@"; }

cleanup() {
  if [ "$KEEP" = "1" ]; then
    # Keep the scrapes too, otherwise there is nothing left to re-diff.
    say ""
    say "containers left up (--keep)"
    say "  scrapes:  $WORK/*.raw  and  $WORK/*.ty"
    say "  re-diff:  diff $WORK/old-c.ty $WORK/new-c.ty"
    say "  teardown: docker compose -f $(pwd)/docker-compose.yml down -v && rm -rf $WORK"
  else
    rm -rf "$WORK"
    compose down -v --remove-orphans >/dev/null 2>&1
  fi
}
trap cleanup EXIT

# --------------------------------------------------------------- feature key
docker info >/dev/null 2>&1 || { say "docker is not running"; exit 1; }

if [ "$COMMUNITY" = "0" ]; then
  for candidate in "${FEATURE_KEY:-}" \
                   "$HOME/.aerospike/features.conf" \
                   "./features.conf" \
                   "$HOME/Downloads/features.conf" \
                   "/etc/aerospike/features.conf"; do
    if [ -n "$candidate" ] && [ -f "$candidate" ]; then
      FEATURE_KEY=$(cd "$(dirname "$candidate")" && pwd)/$(basename "$candidate")
      break
    fi
    FEATURE_KEY=""
  done
fi

if [ "$COMMUNITY" = "1" ] || [ -z "${FEATURE_KEY:-}" ]; then
  # Community Edition: no key needed, but XDR and security do not exist, so
  # aerospike-ce.conf drops feature-key-file and the xdr stanza.
  unset FEATURE_KEY
  export AERO_CONF=aerospike-ce.conf
  export SERVER_A_IMAGE=${SERVER_A_IMAGE:-aerospike/aerospike-server:8.0.0.5}
  export SERVER_B_IMAGE=${SERVER_B_IMAGE:-aerospike/aerospike-server:8.1.2.4}
  export SERVER_C_IMAGE=${SERVER_C_IMAGE:-aerospike/aerospike-server:8.2.0.0}
  say "############################################################"
  if [ "$COMMUNITY" = "1" ]; then
    say "# Community Edition mode (--community)"
  else
    say "# No Enterprise feature key found - falling back to"
    say "# Community Edition. Looked in: \$FEATURE_KEY,"
    say "# ~/.aerospike/, ./, ~/Downloads/, /etc/aerospike/"
  fi
  say "#"
  say "# REDUCED COVERAGE: the aerospike_xdr_* context (40 metric"
  say "# families) and user metrics are Enterprise-only and will"
  say "# NOT be checked. Supply a feature key to cover them."
  say "############################################################"
else
  export FEATURE_KEY
  export AERO_CONF=aerospike.conf
  expiry=$(grep -m1 'valid-until-date' "$FEATURE_KEY" | awk '{print $NF}')
  today=$(date -u +%Y-%m-%d)
  # ISO dates compare correctly as strings, which avoids BSD/GNU date
  # incompatibility. An expired key otherwise surfaces only as an opaque
  # "compose up failed".
  if [ -n "$expiry" ] && [ "$expiry" \< "$today" ]; then
    say "feature key  $FEATURE_KEY"
    say "ERROR: this key EXPIRED on $expiry (today is $today)."
    say "       Aerospike Enterprise will refuse to start. Replace the key,"
    say "       or run with --community to skip Enterprise entirely."
    exit 1
  fi
  say "feature key  $FEATURE_KEY  (valid until ${expiry:-unknown})"
fi

# ------------------------------------------------------- build local exporter
if [ -n "${NEW_EXPORTER_IMAGE:-}" ]; then
  say "new exporter $NEW_EXPORTER_IMAGE (not building locally)"
else
  say "building exporter from $REPO_ROOT"
  if ! docker build -q -t ape-compat:local \
        --build-arg VERSION="$(git -C "$REPO_ROOT" describe --tags --always 2>/dev/null || echo local)" \
        "$REPO_ROOT" >/dev/null; then
    say "exporter build failed"; exit 1
  fi
fi

# --------------------------------------------------- optional server from RPM
# Every server runs on the host platform unless an RPM forces otherwise.
HOST_PLATFORM=linux/$(docker version --format '{{.Server.Arch}}' 2>/dev/null || echo amd64)
export SERVER_C_PLATFORM=$HOST_PLATFORM

if [ -n "$SERVER_RPM" ]; then
  [ -f "$SERVER_RPM" ] || { say "no RPM at $SERVER_RPM"; exit 1; }

  # The RPM's architecture decides the image platform. Installing an x86_64
  # RPM on an arm64 base fails with "does not have a compatible architecture",
  # so detect it from the filename and emulate when it differs from the host.
  case "$SERVER_RPM" in
    *x86_64*|*amd64*)         rpm_platform=linux/amd64 ;;
    *aarch64*|*arm64*)        rpm_platform=linux/arm64 ;;
    *)                        rpm_platform=$HOST_PLATFORM ;;
  esac
  export SERVER_C_PLATFORM=$rpm_platform

  # Validate version ordering BEFORE the build: srv-c is the newest slot and
  # the server axis compares positionally (a -> b -> c), so an older RPM there
  # would report false drops. Checking first avoids a slow pointless build.
  rpm_ver=$(basename "$SERVER_RPM" | sed -n 's/.*enterprise-\([0-9][0-9.]*\)-.*/\1/p')
  b_ver=${SERVER_B_IMAGE##*:}
  if [ -n "$rpm_ver" ] && [ -n "$b_ver" ]; then
    older=$(printf '%s\n%s\n' "$rpm_ver" "$b_ver" | sort -V | head -1)
    if [ "$rpm_ver" != "$b_ver" ] && [ "$older" = "$rpm_ver" ]; then
      say ""
      say "ERROR: RPM version $rpm_ver is OLDER than srv-b ($b_ver), but it would"
      say "       be installed in the newest slot (srv-c). The server axis"
      say "       compares a -> b -> c positionally, so this would report false"
      say "       drops. Override the slot instead, e.g.:"
      say "         SERVER_A_IMAGE=compat-server-rpm:local ./compat-test.sh"
      exit 1
    fi
  else
    say "  warning: could not parse a version from the RPM filename;"
    say "  it will be used as the NEWEST server without an ordering check."
  fi

  say "building server image from $(basename "$SERVER_RPM")  [$rpm_platform]"
  [ -n "$rpm_ver" ] && say "  version: $rpm_ver (newer than srv-b $b_ver)"
  [ "$rpm_platform" != "$HOST_PLATFORM" ] && \
    say "  note: emulating $rpm_platform on $HOST_PLATFORM, this is slow"

  cp "$SERVER_RPM" "$WORK/server.rpm"
  cp Dockerfile.server-rpm "$WORK/Dockerfile"

  # The server RPM ships asd but NOT asinfo, which the compose healthcheck and
  # the sindex-create step both need. Aerospike bundles the tools RPM beside
  # the server RPM, but a standalone download often has none, so search
  # outward: the same directory, then subdirectories, then the parent.
  rpm_dir=$(cd "$(dirname "$SERVER_RPM")" && pwd)
  if [ -z "${TOOLS_RPM:-}" ]; then
    for depth in "$rpm_dir -maxdepth 1" "$rpm_dir -maxdepth 3" "$(dirname "$rpm_dir") -maxdepth 3"; do
      # shellcheck disable=SC2086
      TOOLS_RPM=$(find $depth -name 'aerospike-tools-*.rpm' 2>/dev/null | head -1)
      [ -n "$TOOLS_RPM" ] && break
    done
  fi
  if [ -n "$TOOLS_RPM" ] && [ -f "$TOOLS_RPM" ]; then
    say "  tools:   $(basename "$TOOLS_RPM")"
    cp "$TOOLS_RPM" "$WORK/tools.rpm"
  else
    # Fail now rather than after a slow emulated build and boot that cannot
    # possibly become healthy.
    say ""
    say "ERROR: no aerospike-tools RPM found near $SERVER_RPM"
    say "       The server RPM installs asd but not asinfo, which both the"
    say "       compose healthcheck and sindex creation require, so this"
    say "       run could only fail after a slow build."
    say "       Point at one explicitly:"
    say "         TOOLS_RPM=<path-to-aerospike-tools-*.rpm> \\"
    say "           ./compat-test.sh --server-rpm $SERVER_RPM"
    exit 1
  fi
  if ! docker build --platform "$rpm_platform" -q -t compat-server-rpm:local "$WORK" >/dev/null; then
    say "RPM image build failed; re-running to show the error:"
    docker build --platform "$rpm_platform" -t compat-server-rpm:local "$WORK" 2>&1 \
      | grep -iE 'error|nothing provides|architecture' | head -6 | sed 's/^/    /'
    exit 1
  fi
  export SERVER_C_IMAGE=compat-server-rpm:local
fi

# ------------------------------------------------------------------- bring up
nsrv=$(echo "$SERVERS" | wc -w | tr -d ' ')
nexp=$(echo "$EXPORTERS" | wc -w | tr -d ' ')
say ""
say "starting $nsrv servers and $((nsrv * nexp)) exporters"
# Capture rather than discard: image pull failures, auth errors and bad tags
# are reported by compose itself and are invisible if sent to /dev/null.
if ! compose up -d --wait --wait-timeout 240 >"$WORK/up.log" 2>&1; then
  say "compose up failed."
  say ""
  say "  compose said:"
  grep -iE 'error|fail|unhealthy|not found|denied|pull|manifest|no such' "$WORK/up.log" \
    | head -8 | sed 's/^/    /'
  say ""
  say "  service health:"
  compose ps -a --format '{{.Name}}\t{{.Status}}' 2>&1 | sed 's/^/    /'
  # A dead XDR DC logs connection warnings continuously, so grep for real
  # errors rather than tailing, which would bury the cause.
  for s in $SERVERS; do
    compose logs "srv-$s" 2>&1 \
      | grep -iE 'CRITICAL|FAILED|unknown config|cannot|no such' \
      | head -3 | sed "s/^/    srv-$s  /"
    compose exec -T "srv-$s" sh -c 'command -v asinfo >/dev/null' 2>/dev/null \
      || say "    srv-$s  asinfo MISSING -> healthcheck can never pass"
  done
  say ""
  say "  full output: $WORK/up.log"
  KEEP=1   # preserve the log for inspection
  exit 1
fi
for s in $SERVERS; do
  say "  srv-$s  build=$(compose exec -T "srv-$s" asinfo -v build 2>/dev/null | tr -d '\r')"
done

# -------------------------------------------------------------------- indexes
# No records are written, deliberately. Measured: 695 of 703 metric families
# appear on a completely empty server, because the namespace, node, sets, xdr
# and sindex contexts all report regardless of data, and enable-benchmarks-*
# creates the latency histogram definitions whether or not traffic exists.
#
# The 8 families that need data are all aerospike_latencies_* (write_ms_bucket,
# write_ms_count and their read/batch equivalents). They are given up in
# exchange for dropping the aerospike-tools dependency and halving the runtime.
# This is safe because every comparison is relative: with no server holding
# data, all cells lack the same 8 families. The phantom drops seen earlier came
# from UNEQUAL data between servers, not from its absence.
#
# Indexes are still created. They cost nothing (asinfo ships with the server)
# and they are what makes the sindex context report at all -- and, unexpectedly,
# the sets context too, since registering a set-scoped index registers the set.
say ""
say "creating secondary indexes on all servers"
for s in $SERVERS; do
  compose exec -T "srv-$s" asinfo -v "sindex-create:ns=test;indexname=t_idx;set=testset;indexdata=bin1,numeric" >/dev/null 2>&1
  compose exec -T "srv-$s" asinfo -v "sindex-create:ns=bar;indexname=b_idx;set=barset;indexdata=bin1,numeric"   >/dev/null 2>&1
done
sleep 3

for s in $SERVERS; do
  idx=$(compose exec -T "srv-$s" asinfo -v 'sindex-list:' 2>/dev/null | tr -d '\r' | grep -o 'indexname' | wc -l | tr -d ' ')
  say "  srv-$s  sindexes=$idx"
  [ "$idx" = "0" ] && fail "srv-$s has no secondary index; the sindex context cannot be checked"
done

# --------------------------------------------------------------------- scrape
# Without write traffic, latency histograms only appear once the server has
# published a latency window. Scraping cells one after another therefore reads
# them at different ages: measured, the first cell saw 577 families and the
# last 621, inventing a 44-family "improvement" that does not exist. Reverse
# the order and the same 44 become false DROPS that fail the run.
#
# So: wait for the latency surface to settle, then do a throwaway pass over
# every cell, then the real pass. All cells are then read at the same state.
settle() {
  local prev=-1 now
  for _ in $(seq 1 12); do
    now=$(compose exec -T "exp-old-$(echo "$SERVERS" | awk '{print $1}')" \
            wget -qO- http://localhost:9145/metrics 2>/dev/null \
          | grep -c '^# TYPE aerospike_latencies')
    [ "$now" = "$prev" ] && [ "$now" != "0" ] && { say "  latency surface settled at $now families"; return; }
    prev=$now
    sleep 4
  done
  say "  warning: latency family count still moving ($prev); counts may be noisy"
}

say ""
say "waiting for metric surface to settle"
settle

# Throwaway pass, so no cell is read colder than any other.
for e in $EXPORTERS; do for s in $SERVERS; do
  compose exec -T "exp-$e-$s" wget -qO /dev/null http://localhost:9145/metrics 2>/dev/null
done; done
sleep 3

say ""
say "scraping"
for e in $EXPORTERS; do for s in $SERVERS; do
  cell="$e-$s"
  compose exec -T "exp-$cell" wget -qO- http://localhost:9145/metrics 2>/dev/null > "$WORK/$cell.raw"
  awk '/^# TYPE/ {print $3" "$4}' "$WORK/$cell.raw" | sort -u > "$WORK/$cell.ty"
  up=$(grep -m1 '^aerospike_node_up' "$WORK/$cell.raw" | awk '{print $NF}')
  [ "$up" = "1" ] || fail "exp-$cell could not reach its server (node_up=${up:-missing}); its results are meaningless"
done; done

# Same exporter on the same server must agree across the two passes, and the
# old/new pair on one server must not differ by a whole context.
for s in $SERVERS; do
  lo=$(grep -c '^aerospike_latencies' "$WORK/old-$s.ty")
  ln=$(grep -c '^aerospike_latencies' "$WORK/new-$s.ty")
  [ "$lo" != "$ln" ] && fail "srv-$s latency families differ by exporter ($lo vs $ln); the scrape did not settle, counts are unreliable"
done

# ---------------------------------------------------------- anti-vacuous-green
# Every check in compare() is RELATIVE: it diffs two snapshots and never asks
# whether either one is real. Without the assertions below, a broken parse or
# a misconfiguration that points both cells at the same image produces two
# identical files and reports PASS across the board. That is indistinguishable
# from a genuine clean run -- and on the older servers old and new legitimately
# ARE identical, so it cannot be spotted by eye either.
say ""
say "validating snapshots"

# 1. The two exporters must actually be different images. Compares resolved
#    image IDs, so tag aliasing is caught too.
first_srv=$(echo "$SERVERS" | awk '{print $1}')
old_img=$(docker inspect --format '{{.Image}}' "$(compose ps -q "exp-old-$first_srv")" 2>/dev/null)
new_img=$(docker inspect --format '{{.Image}}' "$(compose ps -q "exp-new-$first_srv")" 2>/dev/null)
if [ -z "$old_img" ] || [ -z "$new_img" ]; then
  fail "could not resolve exporter image IDs; cannot prove the two cells differ"
elif [ "$old_img" = "$new_img" ]; then
  fail "old and new exporters are the SAME image ($old_img). Every comparison
          below is vacuous and would pass no matter what. Check
          NEW_EXPORTER_IMAGE, which is currently '${NEW_EXPORTER_IMAGE:-ape-compat:local}'."
else
  pass "exporters are distinct images"
fi

# 2. Parsed output must be well-formed, not merely non-empty. A format change
#    that shifted awk's $3/$4 would yield equally-garbage files on both sides.
# 3. And large enough to be plausible; 300 is far below any real reading
#    (lowest observed: 577) but far above a broken parse.
MIN_FAMILIES=300
for e in $EXPORTERS; do for s in $SERVERS; do
  f=$WORK/$e-$s.ty
  n=$(wc -l < "$f" | tr -d ' ')
  bad_name=$(awk '$1 !~ /^aerospike_/ {print; exit}' "$f")
  bad_type=$(awk '$2 != "gauge" && $2 != "counter" {print; exit}' "$f")
  if [ -n "$bad_name" ]; then
    fail "$e-$s has a malformed metric name: '$bad_name' -- the # TYPE parse is wrong"
  elif [ -n "$bad_type" ]; then
    fail "$e-$s has a malformed metric type: '$bad_type' -- the # TYPE parse is wrong"
  elif [ "$n" -lt "$MIN_FAMILIES" ]; then
    fail "$e-$s produced only $n families (expected >= $MIN_FAMILIES); the scrape or parse is broken"
  fi
done; done
[ "$FAILED" = "0" ] && pass "all 6 snapshots well-formed and >= $MIN_FAMILIES families"

stats() { printf '%s %s %s' \
  "$(wc -l < "$1" | tr -d ' ')" \
  "$(awk '$2=="gauge"'   "$1" | wc -l | tr -d ' ')" \
  "$(awk '$2=="counter"' "$1" | wc -l | tr -d ' ')"; }

say ""
printf '  %-14s %-10s %-9s %-9s\n' cell families gauges counters
for e in $EXPORTERS; do for s in $SERVERS; do
  read -r f g n <<<"$(stats "$WORK/$e-$s.ty")"
  printf '  %-14s %-10s %-9s %-9s\n' "$e-$s" "$f" "$g" "$n"
done; done

# -------------------------------------------------------------------- compare
compare() {                       # label oldcell newcell
  local label=$1 A=$WORK/$2.ty B=$WORK/$3.ty
  say ""; say "$label"
  if [ ! -s "$A" ] || [ ! -s "$B" ]; then fail "missing scrape data"; return; fi

  read -r fa ga na <<<"$(stats "$A")"
  read -r fb gb nb <<<"$(stats "$B")"

  local dropped g2c c2g lost
  dropped=$(comm -23 <(awk '{print $1}' "$A" | sort -u) <(awk '{print $1}' "$B" | sort -u))
  g2c=$(join -j1 <(sort -k1,1 "$A") <(sort -k1,1 "$B") | awk '$2=="gauge"&&$3=="counter"{print $1}')
  c2g=$(join -j1 <(sort -k1,1 "$A") <(sort -k1,1 "$B") | awk '$2=="counter"&&$3=="gauge"{print $1}' | wc -l | tr -d ' ')

  # 1. names, not counts: a release can add and drop in the same breath
  if [ -z "$dropped" ]; then pass "no metrics dropped"
  else fail "$(echo "$dropped" | grep -c .) metrics DROPPED:"; echo "$dropped" | grep . | sed 's/^/          /'; fi

  # 2. totals and gauges must not shrink; counters legitimately do
  [ "$fb" -ge "$fa" ] && pass "family count $fa -> $fb" || fail "family count DECREASED $fa -> $fb"
  [ "$gb" -ge "$ga" ] && pass "gauge count $ga -> $gb"  || fail "gauge count DECREASED $ga -> $gb"
  info "counter count $na -> $nb (not asserted; counters get promoted to gauges)"

  # 3. gauge -> counter is the regression; the reverse is expected progress
  if [ -z "$g2c" ]; then pass "no gauge reverted to counter"
  else fail "gauge REVERTED to counter:"; echo "$g2c" | sed 's/^/          /'; fi
  [ "$c2g" -gt 0 ] && info "$c2g counters promoted to gauges (expected)"

  # 4. no whole context may vanish
  lost=$(comm -23 \
    <(awk '{print $1}' "$A" | sed 's/^aerospike_//' | awk -F_ '{print $1}' | sort -u) \
    <(awk '{print $1}' "$B" | sed 's/^aerospike_//' | awk -F_ '{print $1}' | sort -u))
  [ -z "$lost" ] && pass "no context went empty" || fail "context(s) went empty: $(echo "$lost" | tr '\n' ' ')"
}

say ""
say "=============== local exporter vs released, same server ==============="
for s in $SERVERS; do compare "on srv-$s:" "old-$s" "new-$s"; done

say ""
say "=============== newer server, same exporter ==============="
prev=""
for s in $SERVERS; do
  if [ -n "$prev" ]; then
    for e in $EXPORTERS; do compare "exporter $e: srv-$prev -> srv-$s:" "$e-$prev" "$e-$s"; done
  fi
  prev=$s
done

say ""
if [ "$FAILED" = "0" ]; then printf '\033[32mALL CHECKS PASSED\033[0m\n'
else printf '\033[31mCOMPATIBILITY REGRESSION DETECTED\033[0m\n'; fi
exit $FAILED
