BINARY   ?= hestia-wails-test
OUTPUT   ?= build/$(BINARY)

.PHONY: all build debug frontend clean run

all: build

frontend:
	cd frontend && bun install && bun run build

build: frontend
	wails build -tags webkit2_41 -o $(OUTPUT)

debug: frontend
	wails build -tags webkit2_41 -devtools -o $(OUTPUT)

dev:
	wails dev -tags webkit2_41

clean:
	rm -rf frontend/node_modules frontend/dist build

run: build
	./$(OUTPUT)
