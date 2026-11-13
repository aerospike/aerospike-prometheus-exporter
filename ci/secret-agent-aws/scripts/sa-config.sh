#!/bin/sh
# Renders /work/sa/config.yaml for the selected transport and backend.
set -eu

transport=${APE_IT_SA_TRANSPORT:-tcp}
backend=${APE_IT_SA_BACKEND:-aws}
out=/work/sa/config.yaml
certs=/work/certs

mkdir -p /work/sa

{
	echo "service:"
	case "$transport" in
	tcp)
		echo "  tcp:"
		echo "    endpoint: 0.0.0.0:3005"
		;;
	tls | mtls)
		echo "  tcp:"
		echo "    endpoint: 0.0.0.0:3005"
		echo "    tls:"
		echo "      cert-file: $certs/sa-server.crt"
		echo "      key-file: $certs/sa-server.key"
		if [ "$transport" = mtls ]; then
			echo "      ca-file: $certs/ca.crt"
		fi
		;;
	uds)
		echo "  uds:"
		echo "    socket-path: /run/sa/secret-agent.sock"
		;;
	*)
		echo "APE_IT_SA_TRANSPORT must be tcp, tls, mtls or uds" >&2
		exit 1
		;;
	esac

	echo ""
	echo "secret-manager:"
	case "$backend" in
	aws)
		echo "  aws:"
		echo "    region: $APE_IT_AWS_REGION"
		echo "    convert-to-base64: false"
		echo "    resources:"
		echo "      APEAgentSecurity: $APE_IT_AWS_SECRET_ARN"
		echo "      APEMissingSecret: $APE_IT_AWS_MISSING_SECRET_ARN"
		;;
	file)
		echo "  file:"
		echo "    convert-to-base64: false"
		echo "    resources:"
		echo "      APEAgentSecurity: /work/sa/secrets.json"
		;;
	esac

	echo ""
	echo "log:"
	echo "  level: info"
} > "$out"

echo "rendered Secret Agent config: transport=$transport backend=$backend"

# A second agent, started only for the plain-text secret. convert-to-base64 applies
# to every key, so this agent cannot share the certificate secret's encoding.
plain_arn=${APE_IT_AWS_PLAIN_SECRET_ARN:-}
if [ "$backend" != file ] && [ -z "$plain_arn" ]; then
	exit 0
fi

case "$transport" in
uds) plain_socket=/run/sa/secret-agent-plain.sock ;;
*) plain_socket= ;;
esac

{
	echo "service:"
	case "$transport" in
	tcp)
		echo "  tcp:"
		echo "    endpoint: 0.0.0.0:3005"
		;;
	tls | mtls)
		echo "  tcp:"
		echo "    endpoint: 0.0.0.0:3005"
		echo "    tls:"
		echo "      cert-file: $certs/sa-server.crt"
		echo "      key-file: $certs/sa-server.key"
		if [ "$transport" = mtls ]; then
			echo "      ca-file: $certs/ca.crt"
		fi
		;;
	uds)
		echo "  uds:"
		echo "    socket-path: $plain_socket"
		;;
	esac

	echo ""
	echo "secret-manager:"
	case "$backend" in
	aws)
		echo "  aws:"
		echo "    region: $APE_IT_AWS_REGION"
		echo "    convert-to-base64: true"
		echo "    resources:"
		echo "      APEPlain: $plain_arn"
		;;
	file)
		echo "  file:"
		echo "    convert-to-base64: true"
		echo "    resources:"
		echo "      APEPlain: /work/sa/plain.json"
		;;
	esac

	echo ""
	echo "log:"
	echo "  level: info"
} > /work/sa/config-plain.yaml

echo "rendered plain-text Secret Agent config: convert-to-base64=true"
