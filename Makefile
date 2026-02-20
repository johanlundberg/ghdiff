.PHONY: test lint fmt check build clean release

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
	go build -o ghdiff .

# Remove build artifacts
clean:
	rm -f ghdiff

# Tag and push a new release (usage: make release v=1.0.0)
release:
ifndef v
	$(error usage: make release v=1.0.0)
endif
	git tag "v$(v)"
	git push origin "v$(v)"
