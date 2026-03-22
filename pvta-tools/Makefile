.PHONY: build clean help test

GO ?= go
BIN_DIR ?= bin
BINARY ?= pvta-tools
PKG ?= .
CGO_ENABLED ?= 0
LD_FLAGS ?= -s -w
GOFLAGS ?=
GOCACHE ?= /tmp/pvta-go-build

help:
	@printf '%s\n' \
		'pvta-tools make targets' \
		'' \
		'  make build' \
		'    Build bin/pvta-tools with the default settings.' \
		'' \
		'  make test' \
		'    Run go test ./... using a writable Go build cache.' \
		'' \
		'  make clean' \
		'    Remove the build output directory.' \
		'' \
		'Variables:' \
		'  BIN_DIR      default: bin' \
		'  BINARY       default: pvta-tools' \
		'  CGO_ENABLED  default: 0' \
		'  GOCACHE      default: /tmp/pvta-go-build' \
		'  GOFLAGS      default: empty'

build:
	mkdir -p $(BIN_DIR)
	GOCACHE=$(GOCACHE) GOFLAGS=$(GOFLAGS) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags "$(LD_FLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

test:
	GOCACHE=$(GOCACHE) GOFLAGS=$(GOFLAGS) $(GO) test ./...

clean:
	rm -rf $(BIN_DIR)
