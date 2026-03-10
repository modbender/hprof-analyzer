BINARY := hprof-analyzer
BIN_DIR := bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test vet clean run snapshot

build:
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) ./cmd/hprof-analyzer/

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(BIN_DIR) dist/

run: build
	$(BIN_DIR)/$(BINARY) $(ARGS)

snapshot:
	goreleaser release --snapshot --clean
