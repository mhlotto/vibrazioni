GO ?= go
BUILD_DIR ?= bin
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=

.PHONY: all build test vet check alias-test install

all: build

build:
	mkdir -p "$(BUILD_DIR)"
	$(GO) build -o "$(BUILD_DIR)/papersplz" ./cmd/papersplz
	ln -sf papersplz "$(BUILD_DIR)/pplz"

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

alias-test: build
	@test "$$($(BUILD_DIR)/papersplz help)" = "$$($(BUILD_DIR)/pplz help)"

check: test vet alias-test

install: build
	mkdir -p "$(DESTDIR)$(BINDIR)"
	install -m 0755 "$(BUILD_DIR)/papersplz" "$(DESTDIR)$(BINDIR)/papersplz"
	ln -sf papersplz "$(DESTDIR)$(BINDIR)/pplz"
