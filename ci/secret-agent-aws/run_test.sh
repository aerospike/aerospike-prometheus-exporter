#!/usr/bin/env bash
# Checks the exit code of run.sh without Docker or AWS. A fake `docker` on PATH stands in for
# docker compose: FAKE_SUITE_FAIL makes the tester step fail, FAKE_SCRUB_FAIL makes every scrub
# fail, and FAKE_SCRUB_FAIL_FIRST makes only the first scrub fail.
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

mkdir "$work/bin"
cat >"$work/bin/docker" <<'EOF'
#!/usr/bin/env bash
args=" $* "
echo "$args" >>"$FAKE_DOCKER_LOG"

case "$args" in
*" setup scrub "*)
	count=$(grep -c ' setup scrub ' "$FAKE_DOCKER_LOG")
	if [[ ${FAKE_SCRUB_FAIL:-0} == 1 ]]; then
		exit 1
	fi
	if [[ ${FAKE_SCRUB_FAIL_FIRST:-0} == 1 && $count -eq 1 ]]; then
		exit 1
	fi
	;;
*" run --rm tester "*)
	if [[ ${FAKE_SUITE_FAIL:-0} == 1 ]]; then
		exit 1
	fi
	;;
esac
exit 0
EOF
chmod +x "$work/bin/docker"

failures=0

# run_case <name> <want exit: zero|nonzero> <want scrub calls> [VAR=value ...]
run_case() {
	local name=$1 want=$2 want_scrubs=$3
	shift 3

	: >"$work/docker.log"
	local code=0
	env "$@" \
		PATH="$work/bin:$PATH" \
		FAKE_DOCKER_LOG="$work/docker.log" \
		APE_IT_SA_BACKEND=aws \
		APE_IT_AWS_SECRET_ARN=arn:aws:secretsmanager:us-east-1:123456789012:secret:ape-test \
		APE_IT_AWS_DIR="$work" \
		APE_IT_SCRUB_ATTEMPTS=3 \
		APE_IT_SCRUB_DELAY=0 \
		bash "$here/run.sh" >"$work/out.log" 2>&1 || code=$?

	local scrubs
	scrubs=$(grep -c ' setup scrub ' "$work/docker.log" || true)

	local ok=1
	if [[ $want == zero && $code -ne 0 ]]; then ok=0; fi
	if [[ $want == nonzero && $code -eq 0 ]]; then ok=0; fi
	if [[ $scrubs -ne $want_scrubs ]]; then ok=0; fi

	if ((ok == 1)); then
		echo "ok   $name (exit $code, $scrubs scrub calls)"
	else
		echo "FAIL $name: exit $code (want $want), $scrubs scrub calls (want $want_scrubs)"
		sed 's/^/     /' "$work/out.log"
		failures=$((failures + 1))
	fi
}

run_case "suite and scrub succeed" zero 1
run_case "suite passes, scrub fails every try" nonzero 3 FAKE_SCRUB_FAIL=1
run_case "suite passes, first scrub try fails, retry works" zero 2 FAKE_SCRUB_FAIL_FIRST=1
run_case "suite fails, scrub works" nonzero 1 FAKE_SUITE_FAIL=1
run_case "suite fails, scrub fails" nonzero 3 FAKE_SUITE_FAIL=1 FAKE_SCRUB_FAIL=1

exit "$failures"
