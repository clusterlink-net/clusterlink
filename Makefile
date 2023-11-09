SW_VERSION ?= latest
IMAGE_ORG ?= mcnet

IMAGE_TAG_BASE ?= quay.io/$(IMAGE_ORG)/mbg
IMG ?= $(IMAGE_TAG_BASE):$(SW_VERSION)
#-----------------------------------------------------------------------------
# Target: clean
#-----------------------------------------------------------------------------
.PHONY: clean
clean: ; $(info cleaning previous builds...)	@
	@rm -rf ./bin

#------------------------------------------------------
# Setup Targets
#------------------------------------------------------

#-- development tooling --
.PHONY: prereqs prereqs-force

prereqs: ; $(info installing dev tooling...) 
	source ./hack/install-devtools.sh

prereqs-force: ; $(info force installing dev tooling...)
	source ./hack/install-devtools.sh --force

.dev-container: Containerfile.dev
	docker build -f Containerfile.dev -t quay.io/$(IMAGE_ORG)/dev:latest .
	touch $@

.PHONY: run-dev-container
run-dev-container: .dev-container
	docker run --rm -it --network bridge \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(CURDIR):$(CURDIR) \
		--workdir $(CURDIR) \
		quay.io/$(IMAGE_ORG)/dev:latest


#-- precommit code checks --
.PHONY: precommit format lint tests-e2e-k8s
precommit: format lint copr-fix
format: fmt
fmt: format-go tidy-go vet-go
vet: vet-go

lint:  ; $(info running linters...)
	@golangci-lint run --config=./.golangci.yaml ./...

tidy-go: ; $(info tidying up go.mod...)
	@go mod tidy

format-go: tidy-go vet-go ; $(info formatting code...)
	@goimports -l -w .

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

build:
	@echo "Start go build phase"
	$(GO) build -o ./bin/gwctl ./cmd/gwctl
	$(GO) build -o ./bin/cl-controlplane ./cmd/cl-controlplane
	$(GO) build -o ./bin/cl-dataplane ./cmd/cl-dataplane
	$(GO) build -o ./bin/cl-go-dataplane ./cmd/cl-go-dataplane
	$(GO) build -o ./bin/cl-adm ./cmd/cl-adm

docker-build: build
	docker build --progress=plain --rm --tag cl-controlplane -f ./cmd/cl-controlplane/Dockerfile .
	docker build --progress=plain --rm --tag cl-dataplane -f ./cmd/cl-dataplane/Dockerfile .
	docker build --progress=plain --rm --tag cl-go-dataplane -f ./cmd/cl-go-dataplane/Dockerfile .
	docker build --progress=plain --rm --tag gwctl -f ./cmd/gwctl/Dockerfile .

build-image:
	docker build --build-arg SW_VERSION="$(SW_VERSION)" -t ${IMG} .
push-image:
	docker push ${IMG}

install:
	cp ./bin/gwctl /usr/local/bin/

clean-tests:
	kind delete cluster --name=mbg1
	kind delete cluster --name=mbg2

#------------------------------------------------------
# Run Targets
#------------------------------------------------------
unit-tests:
	@echo "Running unit tests..."
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
