SHELL=/bin/bash

IMAGE_VERSION ?= latest
IMAGE_ORG ?= clusterlink-net
IMAGE_BASE ?= ghcr.io/$(IMAGE_ORG)

#-----------------------------------------------------------------------------
# Target: clean
#-----------------------------------------------------------------------------
.PHONY: clean
clean: ; $(info cleaning previous builds...)	@
	@rm -rf ./bin

#------------------------------------------------------
# Setup Targets
#------------------------------------------------------

#-- emtpy file directory (used to track build target timestamps --
dist: ; $(info creating dist directory...)
	@mkdir -p $@

#-- development tooling --
.PHONY: prereqs prereqs-force

prereqs: ; $(info installing dev tooling...)
	@. ./hack/install-devtools.sh

prereqs-force: ; $(info force installing dev tooling...)
	@. ./hack/install-devtools.sh --force

.PHONY: dev-container
dev-container: dist/.dev-container

dist/.dev-container: Containerfile.dev | dist ; $(info building dev-container...)
	@docker build -f Containerfile.dev -t $(IMAGE_BASE)/dev:latest .
	@touch $@

.PHONY: run-dev-container
run-dev-container: dev-container ; $(info running dev-container...)
	@docker run --rm -it --network bridge \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(CURDIR):$(CURDIR) \
		--workdir $(CURDIR) \
		$(IMAGE_BASE)/dev:latest


#-- precommit code checks --
.PHONY: precommit format lint tests-e2e-k8s
precommit: format lint copr-fix
format: fmt
fmt: gofumpt format-go tidy-go vet-go
vet: vet-go

lint:  ; $(info running linters...)
	@golangci-lint run --config=./.golangci.yaml ./...

tidy-go: ; $(info tidying up go.mod...)
	@go mod tidy

format-go: tidy-go vet-go ; $(info formatting code...)
	@goimports -l -w .

gofumpt: ; $(info running gofumpt...)
	@gofumpt -l -w .

vet-go: ; $(info vetting code...)
	@go vet ./...

copr-fix: ; $(info adding copyright header...)
	docker run -it --rm -v $(shell pwd):/github/workspace apache/skywalking-eyes header fix

#------------------------------------------------------
# Build targets
#------------------------------------------------------
GO ?= CGO_ENABLED=0 go
# Allow setting of go build flags from the command line.
GOFLAGS :=
BIN_DIR := ./bin
LDFLAGS := -ldflags "-X github.com/clusterlink-net/clusterlink/pkg/versioninfo.GitTag=$(shell git describe --tags --abbrev=0)"
LDFLAGS_OPERATOR:=$(LDFLAGS) \
                -ldflags "-X github.com/clusterlink-net/clusterlink/pkg/versioninfo.Revision=$(shell git rev-parse --short HEAD)"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Controller Gen for crds
CONTROLLER_GEN ?= $(GOBIN)/controller-gen
CONTROLLER_TOOLS_VERSION ?= v0.13.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(GOBIN)
	test -s $(GOBIN)/controller-gen && $(GOBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: codegen
codegen: controller-gen  ## Generate ClusterRole, CRDs and DeepCopyObject.
	$(CONTROLLER_GEN) crd paths="./pkg/apis/..." output:crd:artifacts:config=config/crds/
	$(CONTROLLER_GEN) rbac:roleName=cl-operator-manager-role paths="./pkg/operator/..." output:rbac:dir=config/operator/rbac
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="././pkg/apis/..."
	@goimports -l -w ./pkg/apis/clusterlink.net/v1alpha1/zz_generated.deepcopy.go

cli-build:
	@echo "Start go build phase"
	$(GO) build -o $(BIN_DIR)/gwctl  $(LDFLAGS) ./cmd/gwctl
	$(GO) build -o $(BIN_DIR)/clusterlink $(LDFLAGS) ./cmd/clusterlink

build: cli-build
	$(GO) build -o $(BIN_DIR)/cl-controlplane ./cmd/cl-controlplane
	$(GO) build -o $(BIN_DIR)/cl-dataplane ./cmd/cl-dataplane
	$(GO) build -o $(BIN_DIR)/cl-go-dataplane ./cmd/cl-go-dataplane
	$(GO) build -o $(BIN_DIR)/cl-operator  $(LDFLAGS_OPERATOR) ./cmd/cl-operator/main.go

docker-build: build
	docker build --progress=plain --rm --tag cl-controlplane -f ./cmd/cl-controlplane/Dockerfile .
	docker build --progress=plain --rm --tag cl-dataplane -f ./cmd/cl-dataplane/Dockerfile .
	docker build --progress=plain --rm --tag cl-go-dataplane -f ./cmd/cl-go-dataplane/Dockerfile .
	docker build --progress=plain --rm --tag gwctl -f ./cmd/gwctl/Dockerfile .
	docker build --progress=plain --rm --tag cl-operator -f ./cmd/cl-operator/Dockerfile .

push-image: docker-build
	docker tag cl-dataplane:latest $(IMAGE_BASE)/cl-dataplane:$(IMAGE_VERSION)
	docker push $(IMAGE_BASE)/cl-dataplane:$(IMAGE_VERSION)
	docker tag cl-controlplane:latest $(IMAGE_BASE)/cl-controlplane:$(IMAGE_VERSION)
	docker push $(IMAGE_BASE)/cl-controlplane:$(IMAGE_VERSION)
	docker tag cl-go-dataplane:latest $(IMAGE_BASE)/cl-go-dataplane:$(IMAGE_VERSION)
	docker push $(IMAGE_BASE)/cl-go-dataplane:$(IMAGE_VERSION)
	docker tag gwctl:latest $(IMAGE_BASE)/gwctl:$(IMAGE_VERSION)
	docker push $(IMAGE_BASE)/gwctl:$(IMAGE_VERSION)
	docker tag cl-operator:latest $(IMAGE_BASE)/cl-operator:$(IMAGE_VERSION)
	docker push $(IMAGE_BASE)/cl-operator:$(IMAGE_VERSION)

install:
	mkdir -p ~/.local/bin
	cp ./bin/gwctl ~/.local/bin/

clean-tests:
	kind delete cluster --name=peer1
	kind delete cluster --name=peer2

#------------------------------------------------------
# Run Targets
#------------------------------------------------------
# Envtest use for checking the deployment operator
ENVTEST ?= $(GOBIN)/setup-envtest
ENVTEST_VERSION ?= 395cfc7
ENVTEST_K8S_VERSION = 1.28.0
.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(GOBIN)
	test -s $(GOBIN)/setup-envtest || GOBIN=$(GOBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

unit-tests: envtest
	@echo "Running unit tests..."
	export KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(GOBIN) -p path)";\
	$(GO) test -v -count=1 ./pkg/...  -json -cover | tparse --all

tests-e2e-k8s:
	$(GO) test -p 1 -timeout 30m -v -tags e2e-k8s ./tests/e2e/k8s

run-kind-iperf3:
	python3 demos/iperf3/kind/simple_test.py

run-kind-bookinfo:
	python3 demos/bookinfo/kind/test.py

#------------------------------------------------------
# Clean targets
#------------------------------------------------------
clean-kind:
	kind delete cluster --name=peer1
	kind delete cluster --name=peer2
	kind delete cluster --name=peer3
