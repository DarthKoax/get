SHELL := /usr/bin/env bash -euo pipefail

.PHONY: build build-static build-server run-server run-static clean docker-static docker-server db-to-json

build-static:
	@echo "Building static mode..."
	@cp shortcuts.json internal/database/shortcuts.json
	GOOS=js GOARCH=wasm go build -o web/main.wasm ./cmd/static
	@if [ -f "$$(go env GOROOT)/lib/wasm/wasm_exec.js" ]; then \
		cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js; \
	elif [ -f "$$(go env GOROOT)/misc/wasm/wasm_exec.js" ]; then \
		cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" web/wasm_exec.js; \
	else \
		echo "ERROR: wasm_exec.js not found in GOROOT" >&2; exit 1; \
	fi
	@echo "Static build complete: web/main.wasm"

build-server:
	@echo "Building server mode..."
	@cp shortcuts.json internal/database/shortcuts.json
	go build -o bin/get-server ./cmd/server
	@echo "Server build complete: bin/get-server"

build:
	@. ./build.env 2>/dev/null || true; \
	if [ "$$BUILD_MODE" = "server" ]; then \
		$(MAKE) build-server; \
	else \
		$(MAKE) build-static; \
	fi

run-server: build-server
	./bin/get-server

run-static: docker-static
	@echo "Starting static mode container on port 8080..."
	@docker run --rm -p 8080:80 get:static

run-static-detached: docker-static
	@echo "Starting static mode container in background on port 8080..."
	@docker run -d --name get-static -p 8080:80 get:static
	@echo "Container started. View logs: docker logs get-static"
	@echo "Stop: docker stop get-static && docker rm get-static"

clean:
	rm -f web/main.wasm web/wasm_exec.js bin/get-server

docker-static:
	docker build --target static -t get:static .

docker-server:
	docker build --target server -t get:server .

db-to-json:
	@bash scripts/db-to-json.sh
