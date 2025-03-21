# Variables required for this Makefile
# Include all common variables
include Makefile.vars

# Builds exporter binary
.PHONY: exporter
exporter:
	@echo $(GO_FIPS)
	$(GO_ENV_VARS) GOEXPERIMENT=$(GO_FIPS) go build -ldflags="-X 'main.version=$(VERSION)'" -o aerospike-prometheus-exporter ./cmd

.PHONY: fipsparam
fipsparam: 
ifeq ($(APE_SUPPORTED_OS),validfipsos)
	@echo  "Setting FIPS required params"
	$(eval GO_FIPS=$(GO_BORINGCRYPTO))
	$(eval PKG_FILENAME=$(FIPS_PKG_FILENAME))
else
	@echo  "Fips Exporter build is supported only on CentOS 8 or Red Hat 8 versions or have Golang v1.20 and above"
	exit 1
endif

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: lint
lint:
    ## install golangci-lint from https://raw.githubusercontent.com/golangci/golangci-lint , install and include in PATH
    ## Run golangci-lint against code.
	golangci-lint run

.PHONY: test
test: ## Run all the test-cases defined in this folder.
	## for verbose mode use go test -v ./...	
	go test ./...	

# Builds RPM, DEB and TAR packages
# Requires FPM package manager
.PHONY: deb
deb: exporter vulnerability-scan
	$(MAKE) -C $(ROOT_DIR)/pkg/ deb 

.PHONY: fips-deb
fips-deb: fipsparam exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ fips-deb 

.PHONY: rpm vulnerability-scan
rpm: exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ rpm 

.PHONY: fips-rpm vulnerability-scan
fips-rpm: fipsparam exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ fips-rpm

.PHONY: tar
tar: exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ tar 

.PHONY: fips-tar
fips-tar: fipsparam exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ fips-tar 

# Clean up
.PHONY: clean
clean:
	rm -rf aerospike-prometheus-exporter
	$(MAKE) -C $(ROOT_DIR)/pkg/ $@

# vulnerability scan
.PHONY: vulnerability-scan
vulnerability-scan:
	snyk test --all-projects --policy-path=$(ROOT_DIR)/.snyk --severity-threshold=high

# snyk container 
.PHONY: vulnerability-scan-container
vulnerability-scan-container: vulnerability-scan create-docker-multi-arch
	snyk container test aerospike/aerospike-prometheus-exporter:latest --policy-path=$(ROOT_DIR)/.snyk --file=Dockerfile --severity-threshold=high

# Builds exporter docker image
# Requires docker
.PHONY: docker
docker:
	docker build --build-arg VERSION=$(VERSION) . -t aerospike/aerospike-prometheus-exporter:latest

# NOTE: this builds and pushes the image to aerospike/aerospike-prometheus-exporter docker hub repository
# Use this target only for release
.PHONY: release-docker-multi-arch
release-docker-multi-arch: vulnerability-scan-container
	docker buildx build --build-arg VERSION=$(VERSION) --platform $(DOCKER_MULTI_ARCH_PLATFORMS) --push . -t aerospike/aerospike-prometheus-exporter:latest -t aerospike/aerospike-prometheus-exporter:$(VERSION)

.PHONY: create-docker-multi-arch
create-docker-multi-arch:
	docker buildx build --build-arg VERSION=$(VERSION) --platform linux/amd64 --load . -t aerospike/aerospike-prometheus-exporter:latest

.PHONY: package-linux-arm64
package-linux-arm64:
	$(MAKE) deb rpm tar GOOS=linux GOARCH=arm64 DEB_PKG_ARCH=arm64 ARCH=aarch64

.PHONY: package-linux-amd64
package-linux-amd64:
	$(MAKE) deb rpm tar GOOS=linux GOARCH=amd64 DEB_PKG_ARCH=amd64 ARCH=x86_64
