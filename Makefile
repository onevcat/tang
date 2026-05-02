BINARY := tang
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)

.PHONY: build test lint install clean

build:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/$(BINARY) ./cmd/tang

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" ./cmd/tang

clean:
	rm -rf bin
