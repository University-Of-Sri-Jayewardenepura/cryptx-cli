.PHONY: build release run tools tidy clean

BINARY := cryptx-cli
DIST := dist
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64
# netgo/osusergo avoid libc lookups; timetzdata embeds timezone DB into the binary.
GO_BUILD_FLAGS := -trimpath -tags "netgo osusergo timetzdata" -ldflags "-s -w"

## build: compile the CLI binary
build:
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o $(BINARY) .

## release: build portable binaries for all supported OS/ARCH pairs
release: clean
	@mkdir -p $(DIST)
	@set -e; \
	for platform in $(PLATFORMS); do \
		GOOS="$${platform%/*}"; \
		GOARCH="$${platform#*/}"; \
		ext=""; \
		if [ "$$GOOS" = "windows" ]; then ext=".exe"; fi; \
		out="$(DIST)/$(BINARY)-$$GOOS-$$GOARCH$$ext"; \
		echo "Building $$out"; \
		CGO_ENABLED=0 GOOS="$$GOOS" GOARCH="$$GOARCH" go build $(GO_BUILD_FLAGS) -o "$$out" .; \
	done

## run: run the CLI in development mode
run:
	go run .

## tools: install required CLI tools (pop for email compose)
tools:
	go install github.com/charmbracelet/pop@latest
	@echo "✓ pop installed to $(shell go env GOPATH)/bin/pop"
	@echo "  Add $(shell go env GOPATH)/bin to your PATH if it isn't already."

## tidy: tidy go.mod / go.sum
tidy:
	go mod tidy

## clean: remove generated binaries
clean:
	rm -rf $(DIST) $(BINARY)

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
