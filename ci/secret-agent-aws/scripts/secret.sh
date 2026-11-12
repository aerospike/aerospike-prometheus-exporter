#!/bin/sh
# store: writes the generated material to the APEAgentSecurity secret.
# scrub: replaces the AWS secret value after the run.
# Secret contents are never echoed.
set -eu

backend=${APE_IT_SA_BACKEND:-aws}
encoding=${APE_IT_PEM_ENCODING:-base64}
certs=/work/certs
guard_tag=ape-integration-test

case "$encoding" in
base64 | text) ;;
*)
	echo "APE_IT_PEM_ENCODING must be base64 or text" >&2
	exit 1
	;;
esac

# Refuse to write unless the secret is explicitly marked as disposable test data.
require_test_secret() {
	tag=$(aws secretsmanager describe-secret \
		--region "$APE_IT_AWS_REGION" \
		--secret-id "$APE_IT_AWS_SECRET_ARN" \
		--query "Tags[?Key=='$guard_tag'].Value | [0]" \
		--output text)
	if [ "$tag" != "true" ]; then
		echo "refusing to write $APE_IT_AWS_SECRET_ARN: it is not tagged $guard_tag=true" >&2
		exit 1
	fi
}

secret_json() {
	jq -n \
		--arg encoding "$encoding" \
		--rawfile cert "$certs/server.crt" \
		--rawfile key "$certs/server-enc.key" \
		--rawfile key_plain "$certs/server.key" \
		--rawfile ca "$certs/ca.crt" \
		--rawfile pass "$certs/server.pass" \
		--rawfile wrong "$certs/server.wrong-pass" \
		'{cert_file: $cert, key_file: $key, key_file_plain: $key_plain, root_ca: $ca}
		| map_values(if $encoding == "base64" then @base64 else . end)
		+ {key_file_passphrase: ($pass | rtrimstr("\n")),
		   key_file_passphrase_invalid: ($wrong | rtrimstr("\n"))}'
}

store() {
	case "$backend" in
	aws)
		require_test_secret
		tmp=$(mktemp)
		trap 'rm -f "$tmp"' EXIT
		secret_json > "$tmp"
		aws secretsmanager put-secret-value \
			--region "$APE_IT_AWS_REGION" \
			--secret-id "$APE_IT_AWS_SECRET_ARN" \
			--secret-string "file://$tmp" > /dev/null
		echo "stored test material in $APE_IT_AWS_SECRET_ARN ($encoding PEM)"
		;;
	file)
		mkdir -p /work/sa
		(umask 077 && secret_json > /work/sa/secrets.json)
		# Plain text on purpose: the plain agent has convert-to-base64 enabled.
		jq -n '{username: "plain-user", password: "plain-pass-9f3c"}' > /work/sa/plain.json
		echo "stored test material in the file backend ($encoding PEM)"
		;;
	*)
		echo "APE_IT_SA_BACKEND must be aws or file" >&2
		exit 1
		;;
	esac
}

scrub() {
	if [ "$backend" != aws ]; then
		return 0
	fi

	require_test_secret
	aws secretsmanager put-secret-value \
		--region "$APE_IT_AWS_REGION" \
		--secret-id "$APE_IT_AWS_SECRET_ARN" \
		--secret-string '{"scrubbed":"true"}' > /dev/null
	echo "scrubbed $APE_IT_AWS_SECRET_ARN"
}

case "${1:-}" in
store) store ;;
scrub) scrub ;;
*)
	echo "usage: secret.sh store|scrub" >&2
	exit 1
	;;
esac
