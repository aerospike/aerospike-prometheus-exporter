# Variables required for this Makefile
ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
VERSION = $(shell git describe --tags --always)
GO_ENV_VARS =

ifdef GOOS
GO_ENV_VARS = GOOS=$(GOOS)
endif

ifdef GOARCH
GO_ENV_VARS += GOARCH=$(GOARCH)
endif

DOCKER_MULTI_ARCH_PLATFORMS = linux/amd64,linux/arm64

# Builds exporter binary
.PHONY: exporter
exporter:
	$(GO_ENV_VARS) go build -ldflags="-X 'main.version=$(VERSION)'" -o aerospike-prometheus-exporter .

# Builds RPM, DEB and TAR packages
# Requires FPM package manager
.PHONY: deb
deb: exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ $@

.PHONY: rpm
rpm: exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ $@

.PHONY: tar
tar: exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ $@

# Clean up
.PHONY: clean
clean:
	rm -rf aerospike-prometheus-exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ $@

# Builds exporter docker image
# Requires docker
.PHONY: docker
docker:
	docker build --build-arg VERSION=$(VERSION) . -t aerospike/aerospike-prometheus-exporter:latest

# NOTE: this builds and pushes the image to aerospike/aerospike-prometheus-exporter docker hub repository
# Use this target only for release
.PHONY: release-docker-multi-arch
release-docker-multi-arch:
	docker buildx build --build-arg VERSION=$(VERSION) --platform $(DOCKER_MULTI_ARCH_PLATFORMS) --push . -t aerospike/aerospike-prometheus-exporter:latest -t aerospike/aerospike-prometheus-exporter:$(VERSION)

.PHONY: package-linux-arm64
package-linux-arm64:
	$(MAKE) deb rpm tar GOOS=linux GOARCH=arm64 DEB_PKG_ARCH=arm64 ARCH=arm64

.PHONY: package-linux-amd64
package-linux-amd64:
	$(MAKE) deb rpm tar GOOS=linux GOARCH=amd64 DEB_PKG_ARCH=amd64 ARCH=x86_64
