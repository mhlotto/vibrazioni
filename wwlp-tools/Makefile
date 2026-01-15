.PHONY: build clean test

GO ?= go
BIN_DIR ?= bin
BINARY ?= wwlp
PKG ?= ./cmd/wwlp
CGO_ENABLED ?= 0
LD_FLAGS ?= -s -w
GOFLAGS ?=

build:
	mkdir -p $(BIN_DIR)
	GOFLAGS=$(GOFLAGS) CGO_ENABLED=$(CGO_ENABLED) $(GO) build -trimpath -ldflags "$(LD_FLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

test:
	GOFLAGS=$(GOFLAGS) $(GO) test ./...

clean:
	rm -rf $(BIN_DIR)
