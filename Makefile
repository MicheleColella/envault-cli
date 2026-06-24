BINARY  := envault
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
PREFIX  ?= /usr/local/bin

.PHONY: build clean test lint vet install uninstall

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) ./cmd/envault

clean:
	rm -f $(BINARY)

test:
	go test ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

install: build
	@if [ -f "$(PREFIX)/$(BINARY)" ] && [ "$(FORCE)" != "1" ]; then \
		echo "error: $(PREFIX)/$(BINARY) already exists — run 'make install FORCE=1' to overwrite"; \
		exit 1; \
	fi
	install -m 755 $(BINARY) $(PREFIX)/$(BINARY)
	@echo "installed $(PREFIX)/$(BINARY)"

uninstall:
	@if [ ! -f "$(PREFIX)/$(BINARY)" ]; then \
		echo "nothing to uninstall: $(PREFIX)/$(BINARY) not found"; \
		exit 0; \
	fi
	rm -f $(PREFIX)/$(BINARY)
	@echo "removed $(PREFIX)/$(BINARY)"
