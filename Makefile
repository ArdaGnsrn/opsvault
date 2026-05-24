VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS    := -X github.com/ardagunsuren/opsvault/internal/buildinfo.Version=$(VERSION) \
              -X github.com/ardagunsuren/opsvault/internal/buildinfo.Commit=$(COMMIT) \
              -w -s

.PHONY: build build-linux build-linux-arm64 lint test clean

build:
	go build -ldflags "$(LDFLAGS)" -o dist/opsvault .

build-linux:
	go env -w GOOS=linux GOARCH=amd64 CGO_ENABLED=0
	go build -ldflags "$(LDFLAGS)" -o dist/opsvault-linux-amd64 .
	go env -u GOOS GOARCH CGO_ENABLED

build-linux-arm64:
	go env -w GOOS=linux GOARCH=arm64 CGO_ENABLED=0
	go build -ldflags "$(LDFLAGS)" -o dist/opsvault-linux-arm64 .
	go env -u GOOS GOARCH CGO_ENABLED

lint:
	golangci-lint run ./...

test:
	go test ./... -race -count=1

clean:
	go env -u GOOS GOARCH CGO_ENABLED
	rd /s /q dist 2>nul || rm -rf dist
