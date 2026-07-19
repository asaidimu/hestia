.PHONY: all test test-client test-server version

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
RELEASE ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")

all: build

build:
	go build -v ./cmd/hestia

test:
	go clean -testcache && go test -v ./...

test-server: cmd/test-server/main.go
	go build -o test-server ./cmd/test-server

test-client: test-server
	cd client && bunx vitest --run

version:
	@echo $(VERSION)

release:
	@echo $(RELEASE)
