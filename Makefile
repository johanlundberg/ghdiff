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

# Install man page and shell completions (requires $DESTDIR or $PREFIX)
install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install ghdiff $(DESTDIR)$(PREFIX)/bin/
	install -d $(DESTDIR)$(PREFIX)/share/man/man1
	install -m 644 man/ghdiff.1 $(DESTDIR)$(PREFIX)/share/man/man1/
	install -d $(DESTDIR)$(PREFIX)/share/bash-completion/completions
	install -m 644 completions/ghdiff.bash $(DESTDIR)$(PREFIX)/share/bash-completion/completions/
	install -d $(DESTDIR)$(PREFIX)/share/zsh/site-functions
	install -m 644 completions/_ghdiff $(DESTDIR)$(PREFIX)/share/zsh/site-functions/
	install -d $(DESTDIR)$(PREFIX)/share/fish/vendor_completions.d
	install -m 644 completions/ghdiff.fish $(DESTDIR)$(PREFIX)/share/fish/vendor_completions.d/

# Remove build artifacts
clean:
	rm -f ghdiff

# Tag and push a new release (usage: make release v=1.0.0)
release:
ifndef v
	$(error usage: make release v=1.0.0)
endif
	git tag "v$(v)"
	git push origin main --tags
