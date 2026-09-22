.PHONY: all build test race cover lint fmt vet clean

all: lint test build

## build: build the rdfc-go command into ./bin
build:
	go build -o bin/rdfc-go ./cmd/rdfc-go

## test: run all tests
test:
	go test -count=1 ./...

## race: run all tests with the race detector
race:
	go test -race -count=1 ./...

## cover: run tests and report coverage
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## lint: run formatters, vet and golangci-lint
lint:
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed on:"; gofmt -l .; exit 1; }
	go vet ./...
	golangci-lint run ./...

## fmt: format all Go sources
fmt:
	gofmt -w .

## vet: run go vet
vet:
	go vet ./...

## clean: remove build and coverage artifacts
clean:
	rm -rf bin/ coverage.out coverage.html
