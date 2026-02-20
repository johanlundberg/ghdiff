.PHONY: test lint fmt check build clean

# Run all tests
test:
	go test ./...

# Run linter
lint:
	golangci-lint run ./...

# Run formatter
fmt:
	golangci-lint fmt ./...

# Run all checks (lint + test)
check: lint test

# Build the binary
build:
	go build -o gitdiffview .

# Remove build artifacts
clean:
	rm -f gitdiffview
