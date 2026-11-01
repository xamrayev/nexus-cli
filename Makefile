# Build and check targets for nexus-cli.
#
# Version metadata is stamped into the binary the same way GoReleaser does it
# (see .goreleaser.yaml), so `make build` produces a binary whose
# `nexus version` reports a real version instead of falling back to Go's
# pseudo-version. Override VERSION to build a specific one:
#
#	make build VERSION=v0.3.0-rc.1

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

BIN := bin/nexus

.PHONY: all build run fmt vet test cover smoke check snapshot clean

all: check build

## build: compile bin/nexus with version metadata
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/nexus
	@$(BIN) version

## run: build, then start the local proxy server
run: build
	$(BIN) serve

## fmt: rewrite sources with gofmt
fmt:
	gofmt -w .

## vet: report gofmt drift and run go vet, as CI does
vet:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; echo "$$unformatted"; exit 1; \
	fi
	go vet ./...

## test: run the unit tests
test:
	go test -count=1 ./...

## cover: run the tests with the race detector and print total coverage
cover:
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

## smoke: exercise the CLI and the local server end to end
smoke: build
	scripts/smoke.sh

## check: everything CI runs before a release
check: vet test smoke

## snapshot: build the release artifacts locally, without tagging
snapshot:
	goreleaser release --snapshot --clean

## clean: remove build output
clean:
	rm -rf bin dist coverage.out
