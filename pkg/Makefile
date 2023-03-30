# Include common variables from TOP DIR
include ../Makefile.vars

# Variables required for this Makefile
PKG_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TOP_DIR = $(PKG_DIR)/..
BUILD_DIR = $(PKG_DIR)/build
TARGET_DIR = $(PKG_DIR)/target

# Package variables
NAME = "aerospike-prometheus-exporter"
VERSION = $(shell git describe --tags | cut -c 2-)
MAINTAINER = "Aerospike"
DESCRIPTION = "Aerospike Prometheus Exporter, An agent to export Aerospike metrics to Prometheus"
LICENSE = "Apache License 2.0"
URL = "https://github.com/aerospike/aerospike-prometheus-exporter"
VENDOR = "Aerospike, Inc."
ARCH ?= $(shell uname -m)
DEB_PKG_ARCH ?= $(shell dpkg-architecture -q DEB_BUILD_ARCH)


.PHONY: all
all: deb rpm tar

.PHONY: fipsparam
fipsparam: 
	@echo "fips enabled "$(IS_OS_FIPS_MODE)
	@echo "os-name "$(OS_FULL_NAME)
	@echo "APE_SUPPORTED_OS ==> "$(APE_SUPPORTED_OS)
ifeq ($(APE_SUPPORTED_OS),validfipsos)
	@echo  "Setting FIPS required params"
	$(eval GO_FIPS=$(GO_BORINGCRYPTO))
	$(eval PKG_FILENAME=$(FIPS_PKG_FILENAME))
	@echo  "Current PKG_FILENAME === "$(PKG_FILENAME)
else
	@echo  "Fips Exporter build is supported only on CentOS 8 or Red Hat 8 versions or have Golang v1.20 and above"
	exit 1
endif


.PHONY: fips-deb
fips-deb: fipsparam deb 
	@echo "Completed FIPS Packaging"

.PHONY: deb
deb: prep
	fpm --force \
		--config-files /etc/aerospike-prometheus-exporter \
		--input-type dir \
		--output-type deb \
		--chdir $(BUILD_DIR)/ \
		--after-install $(PKG_DIR)/post-install.sh \
		--name $(NAME) \
		--version $(VERSION) \
		--maintainer $(MAINTAINER) \
		--description $(DESCRIPTION) \
		--license $(LICENSE) \
		--url $(URL) \
		--vendor $(VENDOR) \
		--architecture $(DEB_PKG_ARCH) \
		--package $(TARGET_DIR)/$(PKG_FILENAME)_$(VERSION)_$(DEB_PKG_ARCH).deb


.PHONY: fips-rpm
fips-rpm: fipsparam rpm 
	@echo "Completed FIPS Packaging"

.PHONY: rpm
rpm: prep
	fpm --force \
		--config-files /etc/aerospike-prometheus-exporter \
		--input-type dir \
		--output-type rpm \
		--chdir $(BUILD_DIR)/ \
		--after-install $(PKG_DIR)/post-install.sh \
		--name $(NAME) \
		--version $(VERSION) \
		--maintainer $(MAINTAINER) \
		--description $(DESCRIPTION) \
		--license $(LICENSE) \
		--url $(URL) \
		--vendor $(VENDOR) \
		--architecture $(ARCH) \
		--package $(TARGET_DIR)/$(PKG_FILENAME)-$(VERSION).$(ARCH).rpm

.PHONY: fips-tar
fips-tar: fipsparam tar 

.PHONY: tar
tar: prep
	fpm --force \
		--config-files /etc/aerospike-prometheus-exporter \
		--input-type dir \
		--output-type tar \
		--chdir $(BUILD_DIR)/ \
		--after-install $(PKG_DIR)/post-install.sh \
		--name $(NAME) \
		--version $(VERSION) \
		--maintainer $(MAINTAINER) \
		--description $(DESCRIPTION) \
		--license $(LICENSE) \
		--url $(URL) \
		--vendor $(VENDOR) \
		--package $(TARGET_DIR)/$(PKG_FILENAME)_$(VERSION)_$(ARCH).tar

.PHONY: prep
prep:
	@echo "prep PKG_FILENAME "$(PKG_FILENAME)
	install -d $(TARGET_DIR)
	install -d $(BUILD_DIR)/usr/bin
	install -d $(BUILD_DIR)/etc/aerospike-prometheus-exporter
	install -pm 755 $(TOP_DIR)/aerospike-prometheus-exporter $(BUILD_DIR)/usr/bin/aerospike-prometheus-exporter
	install -pm 644 $(TOP_DIR)/ape.toml $(BUILD_DIR)/etc/aerospike-prometheus-exporter/ape.toml

.PHONY: clean
clean:
	rm -rf $(TARGET_DIR)
	rm -rf $(BUILD_DIR)/etc
	rm -rf $(BUILD_DIR)/usr/bin
