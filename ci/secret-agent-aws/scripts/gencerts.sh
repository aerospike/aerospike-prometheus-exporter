#!/bin/sh
# Generates fresh TLS material for one test run into /work/certs.
# The directory lives on a Docker volume that run.sh removes afterwards.
set -eu

out=/work/certs
rm -rf "$out"
mkdir -p "$out"
cd "$out"
umask 077

new_ca() {
	openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
		-keyout "$1.key" -out "$1.crt" -subj "/CN=$2" \
		-addext "basicConstraints=critical,CA:TRUE" \
		-addext "keyUsage=critical,keyCertSign,cRLSign" 2>/dev/null
}

# new_leaf <name> <common-name> <issuer> <extendedKeyUsage> [subjectAltName]
new_leaf() {
	openssl req -newkey rsa:2048 -nodes \
		-keyout "$1.key" -out "$1.csr" -subj "/CN=$2" 2>/dev/null
	{
		echo "basicConstraints=CA:FALSE"
		echo "keyUsage=critical,digitalSignature,keyEncipherment"
		echo "extendedKeyUsage=$4"
		if [ -n "${5:-}" ]; then
			echo "subjectAltName=$5"
		fi
	} > "$1.ext"
	openssl x509 -req -in "$1.csr" -CA "$3.crt" -CAkey "$3.key" -CAcreateserial \
		-days 1 -out "$1.crt" -extfile "$1.ext" 2>/dev/null
	rm -f "$1.csr" "$1.ext"
}

new_ca ca ape-it-ca
new_ca other-ca ape-it-untrusted-ca

new_leaf server ape-it-exporter ca serverAuth "DNS:localhost,DNS:ape,IP:127.0.0.1"
new_leaf client ape-it-client ca clientAuth
new_leaf rogue-client ape-it-rogue-client other-ca clientAuth
new_leaf sa-server secret-agent ca serverAuth "DNS:secret-agent"

# APE decrypts legacy PEM encryption (Proc-Type/DEK-Info), so the encrypted key uses -traditional.
openssl rand -hex 24 > server.pass
openssl rand -hex 24 > server.wrong-pass
openssl rsa -in server.key -aes256 -traditional \
	-passout file:server.pass -out server-enc.key 2>/dev/null

rm -f ca.key other-ca.key ./*.srl

echo "generated test certificates in $out"
