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


.PHONY: all
all: deb rpm tar

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
		--package $(TARGET_DIR)/aerospike-prometheus-exporter-$(VERSION)-amd64.deb

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
		--package $(TARGET_DIR)/aerospike-prometheus-exporter-$(VERSION)-x86_64.rpm

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
		--package $(TARGET_DIR)/aerospike-prometheus-exporter-$(VERSION).tar

.PHONY: prep
prep:
	cp $(TOP_DIR)/aerospike-prometheus-exporter $(BUILD_DIR)/usr/bin/aerospike-prometheus-exporter
	cp $(TOP_DIR)/ape.toml $(BUILD_DIR)/etc/aerospike-prometheus-exporter/ape.toml

.PHONY: clean
clean:
	rm -rf $(TARGET_DIR)/*
	rm -rf $(BUILD_DIR)/etc/aerospike-prometheus-exporter/ape.toml
	rm -rf $(BUILD_DIR)/usr/bin/aerospike-prometheus-exporter