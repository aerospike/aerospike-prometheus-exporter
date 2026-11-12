#!/bin/sh
set -eu

apk add --no-cache openssl >/dev/null
mkdir -p /certs
cd /certs

openssl req -x509 -newkey rsa:2048 -nodes \
	-keyout ca.key -out ca.crt -days 1 \
	-subj "/CN=ape-test-ca" \
	-addext "basicConstraints=critical,CA:TRUE"

openssl req -x509 -newkey rsa:2048 -nodes \
	-keyout other-ca.key -out other-ca.crt -days 1 \
	-subj "/CN=other-ca" \
	-addext "basicConstraints=critical,CA:TRUE"

cat > /tmp/server.ext <<'EOF'
basicConstraints=CA:FALSE
subjectAltName=DNS:secretagent
extendedKeyUsage=serverAuth
keyUsage=digitalSignature,keyEncipherment
EOF

openssl req -newkey rsa:2048 -nodes \
	-keyout server.key -out server.csr \
	-subj "/CN=secretagent"

openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
	-out server.crt -days 1 -extfile /tmp/server.ext

cat > /tmp/client.ext <<'EOF'
basicConstraints=CA:FALSE
extendedKeyUsage=clientAuth
keyUsage=digitalSignature
EOF

openssl req -newkey rsa:2048 -nodes \
	-keyout client.key -out client.csr \
	-subj "/CN=ape-client"

openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
	-out client.crt -days 1 -extfile /tmp/client.ext

# Secret Agent reads this file from the certs volume. It is not written to the repo.
pem_json() {
	awk 'BEGIN { ORS = "" }
	{
		gsub(/\\/, "\\\\")
		gsub(/"/, "\\\"")
		printf "%s\\n", $0
	}' "$1"
}

{
	printf '{\n'
	printf '  "ca_cert": "%s",\n' "$(pem_json ca.crt)"
	printf '  "client_cert": "%s",\n' "$(pem_json client.crt)"
	printf '  "client_key": "%s"\n' "$(pem_json client.key)"
	printf '}\n'
} > material.json

chmod 644 /certs/*
