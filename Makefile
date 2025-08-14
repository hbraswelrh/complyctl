GO_BUILD_PACKAGES := ./cmd/...
GO_BUILD_BINDIR :=./bin
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG ?= $(shell git tag | sort -V | tail -1 2>/dev/null || echo "v0.0.0")
GIT_TREE_STATE ?= $(shell test -n "`git status --porcelain 2>/dev/null`" && echo "dirty" || echo "clean")
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

GO_LD_EXTRAFLAGS := -X github.com/complytime/complyctl/internal/version.version="$(GIT_TAG)" \
                    -X github.com/complytime/complyctl/internal/version.gitTreeState=$(GIT_TREE_STATE) \
                    -X github.com/complytime/complyctl/internal/version.commit="$(GIT_COMMIT)" \
                    -X github.com/complytime/complyctl/internal/version.buildDate="$(BUILD_DATE)"

MAN_COMPLYCTL = docs/man/complyctl.md
MAN_COMPLYCTL_OUTPUT = docs/man/complyctl.1
MAN_OPENSCAP_PLUGIN = docs/man/complyctl-openscap-plugin.md
MAN_OPENSCAP_PLUGIN_OUTPUT = docs/man/complyctl-openscap-plugin.7
MAN_OPENSCAP_CONF = docs/man/c2p-openscap-manifest.md
MAN_OPENSCAP_CONF_OUTPUT = docs/man/c2p-openscap-manifest.5
MAN_COMPLYCTLPLAN = docs/man/complyctl-plan-edit.md
MAN_COMPLYCTLPLAN_OUTPUT = docs/man/complyctl-plan-edit.2

##@ Compilation

all: clean vendor test-unit build ## compile from scratch
.PHONY: all

build: prep-build-dir ## compile
	go build -mod=vendor -o $(GO_BUILD_BINDIR)/ -ldflags="$(GO_LD_EXTRAFLAGS)" $(GO_BUILD_PACKAGES)
.PHONY: build

##@ Packaging

man: ## generate man pages
	mkdir -p $(dir $(MAN_COMPLYCTL_OUTPUT)) $(dir $(MAN_OPENSCAP_PLUGIN_OUTPUT)) $(dir $(MAN_OPENSCAP_CONF_OUTPUT))
	pandoc -s -t man $(MAN_COMPLYCTL) -o $(MAN_COMPLYCTL_OUTPUT)
	pandoc -s -t man $(MAN_OPENSCAP_PLUGIN) -o $(MAN_OPENSCAP_PLUGIN_OUTPUT)
	pandoc -s -t man $(MAN_OPENSCAP_CONF) -o $(MAN_OPENSCAP_CONF_OUTPUT)
	pandoc -s -t man $(MAN_COMPLYCTLPLAN) -o $(MAN_COMPLYCTLPLAN_OUTPUT)

##@ Environment

dev-setup: dev-setup-commit-hooks ## prepare workspace for contributing
.PHONY: dev-setup

dev-setup-commit-hooks: ## configure pre-commit
	pre-commit install --hook-type pre-commit --hook-type pre-push
.PHONY: dev-setup-commit-hooks

prep-build-dir: ## create build output directory
	mkdir -p ${GO_BUILD_BINDIR}
.PHONY: prep-build-dir

vendor: ## go mod sync
	go mod tidy
	go mod verify
	go mod vendor
.PHONY: vendor

clean:
	@rm -rf ./$(GO_BUILD_BINDIR)/*
	rm -f $(MAN_COMPLYCTL_OUTPUT) $(MAN_OPENSCAP_PLUGIN_OUTPUT) $(MAN_OPENSCAP_CONF_OUTPUT)
.PHONY: clean

##@ Testing

test-unit:
	go test -race -v -coverprofile=coverage.out ./...
.PHONY: test-unit

sanity: vendor format vet ## ensure code is ready for commit
	git diff --exit-code
.PHONY: sanity

format:
	go fmt ./...
.PHONY: format

vet:
	go vet ./...
.PHONY: vet

##@ Help

GREEN := \033[0;32m
TEAL := \033[0;36m
CLEAR := \033[0m

help: ## Show this help.
	@printf "Usage: make $(GREEN)<target>$(CLEAR)\n"
	@awk -v "green=${GREEN}" -v "teal=${TEAL}" -v "clear=${CLEAR}" -F ":.*## *" \
			'/^[a-zA-Z0-9_-]+:/{sub(/:.*/,"",$$1);printf "  %s%-12s%s %s\n", green, $$1, clear, $$2} /^##@/{printf "%s%s%s\n", teal, substr($$1,5), clear}' $(MAKEFILE_LIST)
.PHONY: help
