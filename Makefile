.PHONY: all test test-client test-server version

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
RELEASE ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")

# mattn/go-sqlite3 only compiles the FTS5 module with the sqlite_fts5 tag.
# Full-text indexes (IndexTypeFullText) and TextSearch queries fail at runtime
# without it ("no such module: fts5"), so every sqlite-consuming Go command
# below carries it. Binaries that don't import
# core/persistence/sqlite are unaffected by the tag.
GOTAGS ?= sqlite_fts5

all: build

build:
	go build -tags "$(GOTAGS)" -v ./cmd/hestia

test:
	go clean -testcache && go test -tags "$(GOTAGS)" -v ./...

test-server: cmd/test-server/main.go
	go build -tags "$(GOTAGS)" -o test-server ./cmd/test-server

test-client: test-server
	cd client && bunx vitest --run --fileParallelism=false

version:
	@echo $(VERSION)

release:
	@echo $(RELEASE)
