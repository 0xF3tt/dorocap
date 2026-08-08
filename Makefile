BINARY   := dorocap
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w -X main.version=$(VERSION)
DIST     := dist

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

# Pin build tools so a release can be reproduced after upstream tools change.
GARBLE_VERSION       := v0.13.0
STATICCHECK_VERSION  := 2026.1
GOSEC_VERSION        := v2.28.0
GOVULNCHECK_VERSION  := v1.1.4
CYCLONEDX_VERSION    := v1.7.0

# Random garble seed: a fresh symbol mapping every build, so a dropped or
# recovered binary can't be correlated across releases and no committed seed
# makes the mapping predictable. garble prints the seed it used. Override with
# GARBLE_SEED=<base64> to pin a reproducible mapping (trades away the freshness).
GARBLE_SEED ?= random

.PHONY: help build test vet fmt lint audit install clean release tools tools-check

help:
	@grep -E '^## ' Makefile | sed 's/^## //'

## build: compile a stripped, trimmed, static binary for the host platform
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

## test: run unit tests
test:
	go test ./...

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

## audit: security scan gate (gosec + govulncheck)
audit:
	gosec -quiet ./...
	govulncheck ./...

## install: build and install to GOPATH/bin
install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" .

## clean: remove build artifacts
clean:
	rm -rf $(BINARY) $(DIST)

## tools-check: install pinned lint and security tools
tools-check:
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

## tools: install all pinned dev/CI/release tooling
## golangci-lint isn't reliably `go install`-able; use its installer script
## or brew: https://golangci-lint.run/welcome/install/
tools: tools-check
	go install mvdan.cc/garble@$(GARBLE_VERSION)
	go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@$(CYCLONEDX_VERSION)

## release: run the full check gate, then cross-compile obfuscated binaries
## (via garble) for each platform in PLATFORMS, so a dropped/recovered
## binary doesn't hand a client's blue team readable symbol names for the
## engagement tooling.
release: test lint audit
	@command -v garble >/dev/null || (echo "garble not found: make tools" && exit 1)
	@command -v cyclonedx-gomod >/dev/null || (echo "cyclonedx-gomod not found: make tools" && exit 1)
	@rm -rf $(DIST)
	@mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out=$(DIST)/$(BINARY)-$(VERSION)-$$os-$$arch; \
		[ "$$os" = "windows" ] && out=$$out.exe; \
		echo "building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch garble -tiny -literals -seed=$(GARBLE_SEED) build -trimpath -ldflags "$(LDFLAGS)" -o $$out . || exit 1; \
		done
	@cyclonedx-gomod mod -json -output $(DIST)/sbom.cdx.json .
	@cd $(DIST) && (sha256sum * > SHA256SUMS 2>/dev/null || shasum -a 256 * > SHA256SUMS)
	@cat $(DIST)/SHA256SUMS
