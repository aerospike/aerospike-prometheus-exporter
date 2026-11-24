#!/usr/bin/env bash
# One command for the AWS Secrets Manager TLS/mTLS suite.
#
# AWS backend:
#   APE_IT_AWS_SECRET_ARN   ARN of a disposable secret tagged ape-integration-test=true.
#                           When unset, the suite skips and exits 0 so CI can run it
#                           before the secret is configured.
# Optional:
#   APE_IT_AWS_REGION              defaults to the region in the ARN
#   APE_IT_AWS_MISSING_SECRET_ARN  ARN that does not exist; defaults to one in the same account
#   APE_IT_SA_TRANSPORT            tcp (default), tls, mtls or uds between APE and Secret Agent
#   APE_IT_PEM_ENCODING            base64 (default) or text for PEM values in the secret
#   APE_IT_SA_BACKEND              aws (default) or file to run without AWS
#   APE_IT_AWS_DIR                 directory mounted read-only as ~/.aws; defaults to $HOME/.aws
#   APE_IT_AWS_PLAIN_SECRET_ARN    optional secret with plain-text username and password keys.
#                                  Secret Agent converts those to base64. Unset skips that test.
#   APE_IT_PLAIN_USERNAME          value of the username key in the plain secret
#   APE_IT_PLAIN_PASSWORD          value of the password key in the plain secret
#   APE_IT_SCRUB_ATTEMPTS          tries to scrub the AWS secret after the run; defaults to 3
#   APE_IT_SCRUB_DELAY             seconds between scrub tries; defaults to 5
set -euo pipefail

cd "$(dirname "$0")"

export APE_IT_SA_BACKEND=${APE_IT_SA_BACKEND:-aws}
export APE_IT_SA_TRANSPORT=${APE_IT_SA_TRANSPORT:-tcp}
export APE_IT_PEM_ENCODING=${APE_IT_PEM_ENCODING:-base64}

if [[ $APE_IT_SA_BACKEND == aws && -z ${APE_IT_AWS_SECRET_ARN:-} ]]; then
	echo "APE_IT_AWS_SECRET_ARN is unset; skipping the AWS Secret Agent suite"
	exit 0
fi

if [[ $APE_IT_SA_BACKEND == aws ]]; then

	IFS=: read -r _ partition service region account _ <<<"$APE_IT_AWS_SECRET_ARN"
	if [[ $service != secretsmanager || -z $region || -z $account ]]; then
		echo "APE_IT_AWS_SECRET_ARN is not a Secrets Manager ARN" >&2
		exit 1
	fi

	export APE_IT_AWS_REGION=${APE_IT_AWS_REGION:-$region}
	export APE_IT_AWS_MISSING_SECRET_ARN=${APE_IT_AWS_MISSING_SECRET_ARN:-arn:$partition:secretsmanager:$APE_IT_AWS_REGION:$account:secret:ape-integration-missing-000000}
fi

empty_aws_dir=
if [[ -z ${APE_IT_AWS_DIR:-} ]]; then
	if [[ -d $HOME/.aws ]]; then
		APE_IT_AWS_DIR=$HOME/.aws
	else
		empty_aws_dir=$(mktemp -d)
		APE_IT_AWS_DIR=$empty_aws_dir
	fi
fi
export APE_IT_AWS_DIR

compose=(docker compose -f docker-compose.yml)
stored=0

# scrub_secret replaces the shared AWS secret, retrying a few times. It returns nonzero when
# every try failed, because the generated keys and passphrases would still be in the secret.
scrub_secret() {
	local attempts=${APE_IT_SCRUB_ATTEMPTS:-3}
	local delay=${APE_IT_SCRUB_DELAY:-5}
	local attempt

	for ((attempt = 1; attempt <= attempts; attempt++)); do
		if "${compose[@]}" run --rm --no-deps setup scrub; then
			return 0
		fi

		echo "scrub attempt $attempt of $attempts failed" >&2

		if ((attempt < attempts)); then
			sleep "$delay"
		fi
	done

	return 1
}

cleanup() {
	local status=$?
	local scrub_failed=0

	if ((status != 0)); then
		echo "--- Secret Agent logs" >&2
		"${compose[@]}" logs --no-color secret-agent secret-agent-plain >&2 || true
	fi

	if ((stored == 1)); then
		scrub_secret || scrub_failed=1
	fi

	"${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true

	if [[ -n $empty_aws_dir ]]; then
		rmdir "$empty_aws_dir" || true
	fi

	if ((scrub_failed == 1)); then
		echo "ERROR: could not scrub the test material from the AWS secret; clear it by hand" >&2

		# A passing suite must not hide a failed scrub. A failing suite keeps its own status.
		if ((status == 0)); then
			status=1
		fi
	fi

	if ((status == 0)); then
		echo "AWS Secret Agent TLS/mTLS suite passed"
	else
		echo "AWS Secret Agent TLS/mTLS suite failed (exit $status)" >&2
	fi

	exit "$status"
}
trap cleanup EXIT

echo "backend=$APE_IT_SA_BACKEND transport=$APE_IT_SA_TRANSPORT pem=$APE_IT_PEM_ENCODING"

"${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
"${compose[@]}" build setup
stored=1
"${compose[@]}" run --rm --no-deps setup setup
agents=(secret-agent)
if [[ $APE_IT_SA_BACKEND == file || -n ${APE_IT_AWS_PLAIN_SECRET_ARN:-} ]]; then
	agents+=(secret-agent-plain)
fi
"${compose[@]}" up -d "${agents[@]}"
"${compose[@]}" run --rm tester
