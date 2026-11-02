#!/usr/bin/env bash
# Build the deb, rpm and tgz packages that the shared artifacts CI/CD workflow
# signs and deploys to JFrog, for linux amd64 and arm64.
#
# Required environment:
#   VERSION  package version to publish, for example 1.36.0-pr359-1a2b3c4
#
# Artifacts are written to dist/.
set -euo pipefail

: "${VERSION:?VERSION must be set}"

DIST_DIR=dist
TARGET_DIR=pkg/target
NAME=aerospike-prometheus-exporter

echo "Installing packaging tools"
sudo apt-get update -qq
sudo apt-get install -y -qq ruby ruby-dev build-essential rpm >/dev/null
sudo gem install --no-document fpm >/dev/null
fpm --version

# The shared build job does not set up Go. When the runner has none, install
# the version go.mod asks for; otherwise Go fetches that toolchain itself.
if ! command -v go >/dev/null 2>&1; then
  go_version=$(awk '$1 == "go" {print $2; exit}' go.mod)
  go_root="${RUNNER_TEMP:-/tmp}/go${go_version}"
  echo "Installing Go ${go_version}"
  mkdir -p "${go_root}"
  curl -fsSL "https://go.dev/dl/go${go_version}.linux-$(dpkg --print-architecture).tar.gz" |
    tar -xz -C "${go_root}" --strip-components=1
  export PATH="${go_root}/bin:${PATH}"
fi
go version

echo "Packaging version ${VERSION}"
rm -rf "${TARGET_DIR}" "${DIST_DIR}"
make package-linux-amd64 VERSION="${VERSION}"
make package-linux-arm64 VERSION="${VERSION}"

for tar in "${TARGET_DIR}"/*.tar; do
  gzip -f "${tar}"
  mv "${tar}.gz" "${tar%.tar}.tgz"
done

mkdir -p "${DIST_DIR}"
for file in \
  "${NAME}_${VERSION}-1_amd64.deb" \
  "${NAME}_${VERSION}-1_arm64.deb" \
  "${NAME}-${VERSION}-1.x86_64.rpm" \
  "${NAME}-${VERSION}-1.aarch64.rpm" \
  "${NAME}_${VERSION}_x86_64.tgz" \
  "${NAME}_${VERSION}_aarch64.tgz"; do
  if [[ ! -f "${TARGET_DIR}/${file}" ]]; then
    echo "Expected ${file} in ${TARGET_DIR}, found:" >&2
    ls -l "${TARGET_DIR}" >&2
    exit 1
  fi
  cp "${TARGET_DIR}/${file}" "${DIST_DIR}/"
done

echo "Artifacts in ${DIST_DIR}:"
ls -l "${DIST_DIR}"
