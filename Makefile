PROJECT_NAME := "regoround"
PKG := "github.com/SencilloDev/regoround"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
VERSION := $(shell if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then git describe --exact-match --tags HEAD 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD)"; else echo "dev"; fi)
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

.PHONY: all build docker deps clean test coverage goreleaser fmt lint docs bundle

all: build

deps: ## Get dependencies
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

lint: deps ## Lint the files
	go vet
	gocyclo -over 15 -ignore "generated" ./

test: lint ## Run unittests
	go test -v ./...

coverage: ## Create test coverage report
	go test -cover ./...
	go test ./... -coverprofile=cover.out && go tool cover -html=cover.out -o coverage.html

goreleaser: tidy ## Creates local multiarch releases with GoReleaser
	goreleaser release --snapshot --rm-dist

tidy: ## Pull in dependencies
	go mod tidy && go mod vendor

fmt: ## Format All files
	go fmt ./...

regoroundctl: ## Builds the binary on the current platform
	go build -mod=vendor -a -ldflags "-w -X '$(PKG)/cmd.Version=$(VERSION)'" -o $(PROJECT_NAME)ctl

docs: ## Builds the cli documentation
	mkdir -p docs
	./regoroundctl docs

bundle: ## Creates an OPA bundle
	opa build -b ./bundle -o bundle.tar.gz

clean: ## Reset everything
	docker run --rm -v ./output:/out alpine rm -rf /out/*
	git clean -fd
	git clean -fx
	git reset --hard

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
