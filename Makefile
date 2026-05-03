.PHONY: build test vet lint fmt fmt-check complexity security tidy ci ci-fast ci-tools hooks clean all

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w -X main.version=$(VERSION)
BIN ?= pyhotlint

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/pyhotlint/

test:
	go test ./... -count=1

vet:
	go vet ./...

fmt:
	gofmt -s -w .
	goimports -w .

# fmt-check fails non-zero when any file would be reformatted. Fast
# enough for the pre-commit hook.
fmt-check:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then echo "gofmt would change:"; echo "$$out"; exit 1; fi

lint:
	golangci-lint run

complexity:
	gocyclo -over 12 -ignore '_test\.go$$' .

security:
	gosec ./...

tidy:
	go mod tidy

# Full local-CI parity — same checks GitHub Actions runs.
ci: vet test complexity lint security

# ci-fast is the pre-commit subset: format, vet, build, and tests
# only. ~2 seconds on this repo. Skips golangci-lint (slow startup)
# and gosec (slow + redundant for local edits); pre-push or CI catches
# those.
ci-fast: fmt-check vet build test

# Install the Go-based tools the full CI suite needs. One-time setup
# for a fresh checkout. Equivalent to the bin installs in
# .github/workflows/commit.yml.
ci-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install gotest.tools/gotestsum@latest

# Opt-in pre-commit / pre-push hooks. Points git at the in-repo
# .githooks directory so updates land via PRs alongside code changes.
# Run once per clone; reverse with `git config --unset core.hooksPath`.
hooks:
	git config core.hooksPath .githooks
	@echo "Git hooks installed. Run 'git config --unset core.hooksPath' to disable."

clean:
	rm -f $(BIN) junit-report.xml gosec-report.xml

all: build vet test
