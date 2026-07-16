.PHONY: build release fmt-check test race vet

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf unknown)

build:
	go build -trimpath -o pmusic .

# Keep the release binary at the repository root for ppd compatibility and
# mirror it under dist/ for GitHub release uploads.
release:
	mkdir -p dist
	go build -mod=readonly -trimpath -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o pmusic .
	cp pmusic dist/pmusic

fmt-check:
	test -z "$$(gofmt -l .)"

test:
	go test -count=1 ./...

race:
	go test -race -count=1 ./...

vet:
	go vet ./...
