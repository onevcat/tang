BINARY := tang
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)

.PHONY: build test test-verbose coverage lint install clean

build:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o bin/$(BINARY) ./cmd/tang

test:
	@printf "Test files: "
	@find . -path './.git' -prune -o -name '*_test.go' -print | wc -l | tr -d ' '
	@printf "\nTest cases: "
	@find . -path './.git' -prune -o -name '*_test.go' -print | xargs grep -h '^func Test' | wc -l | tr -d ' '
	@printf "\n"
	go test -cover ./...

test-verbose:
	go test -v ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 1

lint:
	golangci-lint run

install:
	go install -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" ./cmd/tang

clean:
	rm -rf bin
