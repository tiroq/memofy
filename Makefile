.PHONY: build clean test run-core run-ui install quick-install release

BINARY_CORE=bin/memofy-core
BINARY_UI=bin/memofy-ui
GO=CGO_ENABLED=1 GOOS=darwin go
VERSION=0.1.0

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

quick-install:
	bash scripts/quick-install.sh

quick-install-source:
	bash scripts/quick-install.sh --source

release:
	bash scripts/build-release.sh $(VERSION)

