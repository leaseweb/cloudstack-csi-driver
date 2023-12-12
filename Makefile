CMDS=cloudstack-csi-driver cloudstack-csi-sc-syncer

# Revision that gets built into each binary via the main.version
# string. Uses the `git describe` output based on the most recent
# version tag with a short revision suffix or, if nothing has been
# tagged yet, just the revision.
#
# Beware that tags may also be missing in shallow clones as done by
# some CI systems (like TravisCI, which pulls only 50 commits).
REV=$(shell git describe --long --tags --match='v*' --dirty 2>/dev/null || git rev-list -n1 HEAD)

DOCKER?=docker

IMPORTPATH_LDFLAGS = -X main.version=$(REV) 
LDFLAGS = -s -w
FULL_LDFLAGS = $(LDFLAGS) $(IMPORTPATH_LDFLAGS)

.PHONY: all
all: build

.PHONY: build
build: $(CMDS:%=build-%)

.PHONY: container
container: $(CMDS:%=container-%)

.PHONY: clean
clean:
	rm -rf bin test/e2e/e2e.test test/e2e/ginkgo

.PHONY: build-%
$(CMDS:%=build-%): build-%:
	mkdir -p bin 
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '$(FULL_LDFLAGS)' -o "./bin/$*" ./cmd/$*

.PHONY: container-%
$(CMDS:%=container-%): container-%: build-%
	$(DOCKER) build -f ./cmd/$*/Dockerfile -t $*:latest \
		--label org.opencontainers.image.revision=$(REV) .

.PHONY: test
test:
	go test ./...

.PHONY: test-sanity
test-sanity:
	go test --tags=sanity ./test/sanity

.PHONY: setup-external-e2e
setup-external-e2e: test/e2e/e2e.test test/e2e/ginkgo

test/e2e/e2e.test test/e2e/ginkgo:
	curl --location https://dl.k8s.io/v1.27.5/kubernetes-test-linux-amd64.tar.gz | \
		tar --strip-components=3 -C test/e2e -zxf - kubernetes/test/bin/e2e.test kubernetes/test/bin/ginkgo 

.PHONY: test-e2e
test-e2e: setup-external-e2e
	bash ./test/e2e/run.sh
