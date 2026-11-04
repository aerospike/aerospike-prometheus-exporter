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

require_package() {
  if [[ ! -f "${TARGET_DIR}/$1" ]]; then
    echo "Expected $1 in ${TARGET_DIR}, found:" >&2
    ls -l "${TARGET_DIR}" >&2
    exit 1
  fi
}

# The shared deploy step takes each package's target distribution from its file
# name and rejects packages without one. The exporter binary is static, so the
# same package is published under every distribution listed here.
DEB_DISTROS=${DEB_DISTROS:-ubuntu20.04 ubuntu22.04 ubuntu24.04 ubuntu26.04 debian11 debian12 debian13}
RPM_DISTROS=${RPM_DISTROS:-el8 el9 el10}

mkdir -p "${DIST_DIR}"
for arch in amd64 arm64; do
  deb="${NAME}_${VERSION}-1_${arch}.deb"
  require_package "${deb}"
  for distro in ${DEB_DISTROS}; do
    cp "${TARGET_DIR}/${deb}" "${DIST_DIR}/${NAME}_${VERSION}-1_${distro}_${arch}.deb"
  done
done
for arch in x86_64 aarch64; do
  rpm="${NAME}-${VERSION}-1.${arch}.rpm"
  require_package "${rpm}"
  for distro in ${RPM_DISTROS}; do
    cp "${TARGET_DIR}/${rpm}" "${DIST_DIR}/${NAME}-${VERSION}-1.${distro}.${arch}.rpm"
  done

  tgz="${NAME}_${VERSION}_${arch}.tgz"
  require_package "${tgz}"
  cp "${TARGET_DIR}/${tgz}" "${DIST_DIR}/"
done

echo "Artifacts in ${DIST_DIR}:"
ls -l "${DIST_DIR}"
