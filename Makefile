.PHONY: build run test test-coverage test-verbose bench clean install deps lint fmt

BINARY_NAME=actionlint-mcp
INSTALL_PATH=/usr/local/bin
COVERAGE_FILE=coverage.txt

build:
	go build -o $(BINARY_NAME) .

run:
	go run .

test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-verbose:
	go test -v -race -count=1 ./...

bench:
	go test -bench=. -benchmem ./...

clean:
	go clean
	rm -f $(BINARY_NAME) $(COVERAGE_FILE) coverage.html

install: build
	cp $(BINARY_NAME) $(INSTALL_PATH)/
	chmod +x $(INSTALL_PATH)/$(BINARY_NAME)

deps:
	go mod download
	go mod tidy

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		go vet ./...; \
	fi

fmt:
	go fmt ./...
	gofmt -s -w .