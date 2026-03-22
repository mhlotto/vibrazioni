.PHONY: build clean help test

GO ?= go
BIN_DIR ?= bin
BINARY ?= wwlp
PKG ?= ./cmd/wwlp
CGO_ENABLED ?= 0
DEFAULT_LLAMA_HOST ?= 127.0.0.1
LD_FLAGS ?= -s -w -X 'main.defaultLlamaHost=$(DEFAULT_LLAMA_HOST)'
GOFLAGS ?=

help:
	@printf '%s\n' \
		'wwlp-tools make targets' \
		'' \
		'  make build' \
		'    Build bin/wwlp with the default settings.' \
		'' \
		'  make build DEFAULT_LLAMA_HOST=HOST' \
		'    Inject the default llama.cpp host at link time.' \
		'' \
		'  make test' \
		'    Run go test ./... .' \
		'' \
		'  make clean' \
		'    Remove the build output directory.' \
		'' \
		'Variables:' \
		'  DEFAULT_LLAMA_HOST  default: 127.0.0.1' \
		'  BIN_DIR             default: bin' \
		'  BINARY              default: wwlp' \
		'  CGO_ENABLED         default: 0'

build:
	mkdir -p $(BIN_DIR)
	GOFLAGS=$(GOFLAGS) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags "$(LD_FLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

test:
	GOFLAGS=$(GOFLAGS) $(GO) test ./...

clean:
	rm -rf $(BIN_DIR)
