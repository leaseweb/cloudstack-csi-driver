CMDS=cloudstack-csi-driver cloudstack-csi-sc-syncer

PKG=github.com/leaseweb/cloudstack-csi-driver
# Revision that gets built into each binary via the main.version
# string. Uses the `git describe` output based on the most recent
# version tag with a short revision suffix or, if nothing has been
# tagged yet, just the revision.
#
# Beware that tags may also be missing in shallow clones as done by
# some CI systems (like TravisCI, which pulls only 50 commits).
REV=$(shell git describe --long --tags --match='v*' --dirty 2>/dev/null || git rev-list -n1 HEAD)
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u -Iseconds)

DOCKER?=docker

IMPORTPATH_LDFLAGS = -X ${PKG}/pkg/driver.driverVersion=$(REV) -X ${PKG}/pkg/driver.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/driver.buildDate=${BUILD_DATE}
LDFLAGS = -s -w
FULL_LDFLAGS = $(LDFLAGS) $(IMPORTPATH_LDFLAGS)

export REPO_ROOT := $(shell git rev-parse --show-toplevel)

# Directories
TOOLS_DIR := $(REPO_ROOT)/hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
BIN_DIR ?= bin

GO_INSTALL := ./hack/go_install.sh

GOLANGCI_LINT_BIN := golangci-lint
GOLANGCI_LINT_VER := v2.7.2
GOLANGCI_LINT := $(abspath $(TOOLS_BIN_DIR)/$(GOLANGCI_LINT_BIN)-$(GOLANGCI_LINT_VER))
GOLANGCI_LINT_PKG := github.com/golangci/golangci-lint/v2/cmd/golangci-lint

MOCKGEN_BIN := mockgen
MOCKGEN_VER := v0.6.0
MOCKGEN := $(abspath $(TOOLS_BIN_DIR)/$(MOCKGEN_BIN)-$(MOCKGEN_VER))
MOCKGEN_PKG := go.uber.org/mock/mockgen

##@ Linting
## --------------------------------------
## Linting
## --------------------------------------

.PHONY: fmt
fmt: ## Run go fmt on the whole project.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet on the whole project.
	go vet ./...

.PHONY: lint
lint: $(GOLANGCI_LINT) generate-mocks ## Run linting for the project.
	$(MAKE) fmt
	$(MAKE) vet
	$(GOLANGCI_LINT) run -v --timeout 360s ./...

##@ Build
## --------------------------------------
## Build
## --------------------------------------

.PHONY: all
all: build

.PHONY: build
build: $(CMDS:%=build-%)

.PHONY: container
container: $(CMDS:%=container-%)

.PHONY: clean
clean:
	rm -rf $(TOOLS_BIN_DIR)
	rm -rf bin test/e2e/e2e.test test/e2e/ginkgo

.PHONY: build-%
$(CMDS:%=build-%): build-%:
	mkdir -p bin 
	CGO_ENABLED=0 go build -ldflags '$(FULL_LDFLAGS)' -o "./bin/$*" ./cmd/$*

.PHONY: container-%
$(CMDS:%=container-%): container-%: build-%
	$(DOCKER) build -f ./cmd/$*/Dockerfile -t $*:latest \
		--label org.opencontainers.image.revision=$(REV) .

.PHONY: generate-mocks
generate-mocks: $(MOCKGEN) pkg/cloud/mock_cloud.go ## Generate mocks needed for testing. Primarily mocks of the cloud package.
pkg/cloud/mock%.go: $(shell find ./pkg/cloud -type f -name "*test*" -prune -o -print)
	go generate ./...

.PHONY: test
test:
	go test ./...

.PHONY: test-sanity
test-sanity:
	go test --tags=sanity ./test/sanity

.PHONY: setup-external-e2e
setup-external-e2e: test/e2e/e2e.test test/e2e/ginkgo

test/e2e/e2e.test test/e2e/ginkgo:
	curl --location https://dl.k8s.io/v1.34.2/kubernetes-test-linux-amd64.tar.gz | \
		tar --strip-components=3 -C test/e2e -zxf - kubernetes/test/bin/e2e.test kubernetes/test/bin/ginkgo 

.PHONY: test-e2e
test-e2e: setup-external-e2e
	bash ./test/e2e/run.sh

##@ hack/tools:

.PHONY: $(GOLANGCI_LINT_BIN)
$(GOLANGCI_LINT_BIN): $(GOLANGCI_LINT) ## Build a local copy of golangci-lint.

.PHONY: $(MOCKGEN_BIN)
$(MOCKGEN_BIN): $(MOCKGEN) ## Build a local copy of mockgen.

$(GOLANGCI_LINT): # Build golangci-lint from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) $(GOLANGCI_LINT_PKG) $(GOLANGCI_LINT_BIN) $(GOLANGCI_LINT_VER)

$(MOCKGEN): # Build mockgen from tools folder.
	GOBIN=$(TOOLS_BIN_DIR) $(GO_INSTALL) $(MOCKGEN_PKG) $(MOCKGEN_BIN) $(MOCKGEN_VER)
