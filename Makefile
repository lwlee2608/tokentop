GO = $(shell which go 2>/dev/null)

APP             := tokentop
VERSION         ?= v0.2.1
LDFLAGS         := -ldflags "-X main.AppVersion=$(VERSION)"
PREFIX          := $(HOME)/.local

.PHONY: all build clean run test install

all: clean build

clean:
	$(GO) clean -testcache
	$(RM) -rf bin/*
build:
	$(GO) build -o bin/$(APP) $(LDFLAGS) cmd/$(APP)/*.go
run:
	$(GO) run $(LDFLAGS) cmd/$(APP)/*.go
test:
	$(GO) test -v ./...
install: build
	install -d $(PREFIX)/bin
	install -m 755 bin/$(APP) $(PREFIX)/bin/$(APP)
