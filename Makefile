.PHONY: build clean test run-core run-ui install

BINARY_CORE=bin/memofy-core
BINARY_UI=bin/memofy-ui
GO=CGO_ENABLED=1 GOOS=darwin go

build:
	mkdir -p bin
	$(GO) build -o $(BINARY_CORE) cmd/memofy-core/main.go
	$(GO) build -o $(BINARY_UI) cmd/memofy-ui/main.go

clean:
	rm -rf bin/
	rm -rf ~/.cache/memofy/

test:
	go test -v ./internal/...
	go test -v ./tests/integration/...

run-core:
	$(BINARY_CORE)

run-ui:
	$(BINARY_UI)

install:
	./scripts/install-launchagent.sh
