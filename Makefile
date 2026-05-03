.PHONY: build test vet lint fix clean ci all

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o reprobundle ./cmd/reprobundle/

test:
	go test ./... -count=1

vet:
	go vet ./...

lint:
	golangci-lint run

fix:
	golangci-lint run --fix

clean:
	rm -f reprobundle

ci: build vet test

all: build
