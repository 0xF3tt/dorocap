BINARY   := dorocap
# 0.0.1 until the first v* tag exists. A tag such as v0.0.1 replaces this.
VERSION  := $(shell git describe --tags --match 'v*' --dirty 2>/dev/null || echo 0.0.1)
LDFLAGS  := -s -w -X main.version=$(VERSION)
DIST     := dist

# `make tools-check` installs into GOPATH/bin; find those tools even when the
# calling shell (or a pre-commit hook) has no GOPATH/bin on its PATH.
export PATH := $(shell go env GOPATH)/bin:$(PATH)

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

# Tool versions and checksums live in tools/go.mod. golangci-lint is pinned in
# the GitHub Actions workflows because `go install` is not a supported install path.

.PHONY: help build test test-race vet fmt lint audit install clean release tools tools-check

help:
	@grep -E '^## ' Makefile | sed 's/^## //'

## build: compile a stripped, trimmed, static binary for the host platform
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: run unit tests
test:
	go test ./...

## test-race: run unit tests with the race detector
test-race:
	go test -race ./...

## vet: go vet
vet:
	go vet ./...

## fmt: check formatting (fails if gofmt would change anything)
fmt:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt-ed:"; echo "$$unformatted"; exit 1; \
	fi

## lint: full static-analysis gate (fmt + vet + staticcheck + golangci-lint)
lint: fmt vet
	staticcheck ./...
	golangci-lint run ./...

## audit: security scan gate (gitleaks + gosec + govulncheck)
audit:
	gitleaks detect --redact --no-git --source .
	gosec -quiet ./...
	govulncheck ./...

## install: build and install to GOPATH/bin
install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" .

## clean: remove build artifacts
clean:
	rm -rf $(BINARY) $(DIST)

## tools-check: install pinned lint, scan, and SBOM tools from tools/go.mod
tools-check:
	cd tools && go install tool

## tools: install all pinned dev/CI/release tooling
## golangci-lint isn't reliably `go install`-able; use its installer script
## or brew: https://golangci-lint.run/welcome/install/
tools: tools-check

## release: run the full check gate, then cross-compile trimmed static binaries.
## -buildvcs=false keeps the binary free of the local checkout path. The
## CycloneDX file is a source SBOM, not a description of the compiled binary.
release: test-race lint audit
	@command -v cyclonedx-gomod >/dev/null || (echo "cyclonedx-gomod not found: make tools" && exit 1)
	@rm -rf $(DIST)
	@mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out=$(DIST)/$(BINARY)-$(VERSION)-$$os-$$arch; \
		[ "$$os" = "windows" ] && out=$$out.exe; \
		echo "building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" -o $$out . || exit 1; \
		done
	@cyclonedx-gomod mod -json -output $(DIST)/sbom.cdx.json .
	@cd $(DIST) && (sha256sum * > SHA256SUMS 2>/dev/null || shasum -a 256 * > SHA256SUMS)
	@cat $(DIST)/SHA256SUMS
