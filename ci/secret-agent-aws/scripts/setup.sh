#!/bin/sh
set -eu

backend=${APE_IT_SA_BACKEND:-aws}

if [ "$backend" = aws ]; then
	: "${APE_IT_AWS_SECRET_ARN:?APE_IT_AWS_SECRET_ARN is required}"
	: "${APE_IT_AWS_REGION:?APE_IT_AWS_REGION is required}"
	: "${APE_IT_AWS_MISSING_SECRET_ARN:?APE_IT_AWS_MISSING_SECRET_ARN is required}"
fi

case "${1:-setup}" in
setup)
	/bin/sh /scripts/gencerts.sh
	/bin/sh /scripts/secret.sh store
	/bin/sh /scripts/sa-config.sh
	;;
scrub)
	/bin/sh /scripts/secret.sh scrub
	;;
*)
	echo "usage: setup.sh setup|scrub" >&2
	exit 1
	;;
esac
