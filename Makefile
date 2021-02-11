SHELL := /usr/bin/env bash -o pipefail

# This controls the location of the cache.
PROJECT := kps-connector-go-sdk
# This controls the remote HTTPS git location to compare against for breaking changes in CI.
#
# Most CI providers only clone the branch under test and to a certain depth, so when
# running buf breaking in CI, it is generally preferable to compare against
# the remote repository directly.
#
# Basic authentication is available, see https://buf.build/docs/inputs#https for more details.
HTTPS_GIT := https://github.com/nutanix/kps-connector-go-sdk.git
# This controls the remote SSH git location to compare against for breaking changes in CI.
#
# CI providers will typically have an SSH key installed as part of your setup for both
# public and private repositories. Buf knows how to look for an SSH key at ~/.ssh/id_rsa
# and a known hosts file at ~/.ssh/known_hosts or /etc/ssh/known_hosts without any further
# configuration. We demo this with CircleCI.
#
# See https://buf.build/docs/inputs#ssh for more details.
SSH_GIT := ssh://git@github.com/nutanix/kps-connector-go-sdk.git
# This controls the version of buf to install and use.
BUF_VERSION := 0.37.0
# If true, Buf is installed from source instead of from releases
BUF_INSTALL_FROM_SOURCE := false

### Everything below this line is meant to be static, i.e. only adjust the above variables. ###

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)
# Buf will be cached to ~/.cache/buf-example.
CACHE_BASE := $(HOME)/.cache/$(PROJECT)
# This allows switching between i.e a Docker container and your local setup without overwriting.
CACHE := $(CACHE_BASE)/$(UNAME_OS)/$(UNAME_ARCH)
# The location where buf will be installed.
CACHE_BIN := $(CACHE)/bin
# Marker files are put into this directory to denote the current version of binaries that are installed.
CACHE_VERSIONS := $(CACHE)/versions
PROTOC_GEN_GO := $(CACHE)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(CACHE)/bin/protoc-gen-go-grpc
GO_FILES := $(shell \
	find . '(' -path '*/.*' -o -path './vendor' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

# Update the $PATH so we can use buf directly
export PATH := $(abspath $(CACHE_BIN)):$(PATH)
# Update GOBIN to point to CACHE_BIN for source installations
export GOBIN := $(abspath $(CACHE_BIN))
# This is needed to allow versions to be added to Golang modules with go get
export GO111MODULE := on

# If $GOPATH/bin/protoc-gen-go does not exist, we'll run this command to install it.
$(PROTOC_GEN_GO):
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/protobuf/cmd/protoc-gen-go

# If $GOPATH/bin/protoc-gen-go-grpc does not exist, we'll run this command to install it.
$(PROTOC_GEN_GO_GRPC):
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

GOLINT = $(GOBIN)/golint
GOSEC = $(GOBIN)/gosec

$(GOLINT):
	go install golang.org/x/lint/golint

$(GOSEC):
	go install github.com/securego/gosec/v2/cmd/gosec

# BUF points to the marker file for the installed version.
#
# If BUF_VERSION is changed, the binary will be re-downloaded.
BUF := $(CACHE_VERSIONS)/buf/$(BUF_VERSION)
$(BUF):
	@rm -f $(CACHE_BIN)/buf
	@mkdir -p $(CACHE_BIN)
ifeq ($(BUF_INSTALL_FROM_SOURCE),true)
	$(eval BUF_TMP := $(shell mktemp -d))
	cd $(BUF_TMP); go get github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	@rm -rf $(BUF_TMP)
else
	curl -sSL \
		"https://github.com/bufbuild/buf/releases/download/v$(BUF_VERSION)/buf-$(UNAME_OS)-$(UNAME_ARCH)" \
		-o "$(CACHE_BIN)/buf"
	chmod +x "$(CACHE_BIN)/buf"
endif
	@rm -rf $(dir $(BUF))
	@mkdir -p $(dir $(BUF))
	@touch $(BUF)

.DEFAULT_GOAL := local

# deps allows us to install deps without running any checks.

.PHONY: deps
deps: $(BUF)

.PHONY: generate
generate:
	buf generate

.PHONY: build
build: generate
	GOOS=linux GOARCH=386 go build ./...

.PHONY: install
install:
	go mod download

.PHONY: lint
lint: $(GOLINT) $(GOSEC)
	rm -rf lint.log
	echo "Checking protobuf linting..."
	buf lint  2>&1 | tee lint.log
	#buf breaking --against '.git#branch=main'
	echo "Checking formatting..."
	gofmt -d -s $(GO_FILES) 2>&1 | tee lint.log
	echo "Checking vet..."
	go vet ./... 2>&1 | tee -a lint.log
	echo "Checking lint..."
	$(GOLINT) ./... 2>&1 | tee -a lint.log
	echo "Checking security vulnerabilities..."
	$(GOSEC) -quiet ./... 2>&1 | tee -a lint.log

.PHONY: test
test:
	go test -v ./...

.PHONY: cover
cover:
	go test -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

# local is what we run when testing locally.
# This does breaking change detection against our local git repository.

.PHONY: local
local: $(BUF) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	make generate
	make build
	make install
	make lint
	make test
	make cover

# https is what we run when testing in most CI providers.
# This does breaking change detection against our remote HTTPS git repository.

.PHONY: https
https: $(BUF) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	make generate
	make build
	make install
	make lint
	make test
	make cover

# ssh is what we run when testing in CI providers that provide ssh public key authentication.
# This does breaking change detection against our remote HTTPS ssh repository.
# This is especially useful for private repositories.

.PHONY: ssh
ssh: $(BUF) $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	make generate
	make build
	make install
	make lint
	make test
	make cover

# clean deletes any files not checked in and the cache for all platforms.

.PHONY: clean
clean:
	git clean -xdf
	rm -rf $(CACHE_BASE)

# For updating this repository

.PHONY: updateversion
updateversion:
ifndef VERSION
	$(error "VERSION must be set")
else
ifeq ($(UNAME_OS),Darwin)
	sed -i '' "s/BUF_VERSION := [0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/BUF_VERSION := $(VERSION)/g" Makefile
else
	sed -i "s/BUF_VERSION := [0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/BUF_VERSION := $(VERSION)/g" Makefile
endif
endif

# For updating the kps-connector-idl submodule

.PHONY: updatesubmodule
updatesubmodule:
	git submodule foreach git pull origin main
