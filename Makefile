BINARY   := dorocap
# 0.0.1 until the first v* tag exists. A tag such as v0.0.1 replaces this.
VERSION  := $(shell git describe --tags --match 'v*' --dirty 2>/dev/null || echo 0.0.1)
LDFLAGS  := -s -w -X main.version=$(VERSION)
DIST     := dist

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

# Scanner versions. `go run pkg@version` builds each tool from its own tested
# go.mod and verifies it against sum.golang.org, so no tools module is needed.
# Keep GITLEAKS_VERSION equal to the gitleaks rev in .pre-commit-config.yaml.
# golangci-lint (which also runs staticcheck and gosec) is pinned in the
# workflows; install the same version locally with brew or mise.
GITLEAKS_VERSION    := v8.30.1
GOVULNCHECK_VERSION := v1.9.0
CYCLONEDX_VERSION   := v1.12.0

GITLEAKS    := go run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION)
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
CYCLONEDX   := go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@$(CYCLONEDX_VERSION)

.PHONY: help build test test-race vet fmt lint audit install clean release

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

## vet: go vet (its stdversion check flags APIs newer than go.mod allows)
vet:
	go vet ./...

## fmt: check formatting (fails if gofmt would change anything)
fmt:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt-ed:"; echo "$$unformatted"; exit 1; \
	fi

## lint: static-analysis gate (fmt + vet + golangci-lint with staticcheck and gosec)
lint: fmt vet
	golangci-lint run ./...

## audit: security scan gate (gitleaks over the git history + govulncheck)
audit:
	$(GITLEAKS) detect --redact --source .
	$(GOVULNCHECK) ./...

## install: build and install to GOPATH/bin
install:
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" .

## clean: remove build artifacts
clean:
	rm -rf $(BINARY) $(DIST)

## release: run the full check gate, then cross-compile trimmed static binaries.
## -buildvcs=false keeps the binary free of the local checkout path. The
## CycloneDX file is a source SBOM, not a description of the compiled binary.
release: test-race lint audit
	@rm -rf $(DIST)
	@mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out=$(DIST)/$(BINARY)-$(VERSION)-$$os-$$arch; \
		[ "$$os" = "windows" ] && out=$$out.exe; \
		echo "building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -buildvcs=false -ldflags "$(LDFLAGS)" -o $$out . || exit 1; \
		done
	@$(CYCLONEDX) mod -json -output $(DIST)/sbom.cdx.json .
	@cd $(DIST) && (sha256sum * > SHA256SUMS 2>/dev/null || shasum -a 256 * > SHA256SUMS)
	@cat $(DIST)/SHA256SUMS
