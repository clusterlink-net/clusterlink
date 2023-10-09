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

prereqs: 											## Verify that required utilities are installed
	@echo -- $@ for MBG Project--
	@go version || (echo "Please install GOLANG: https://go.dev/doc/install" && exit 1)

test-prereqs: prereqs
	@which goimports || (echo "Please install goimports: https://pkg.go.dev/golang.org/x/tools/cmd/goimports" && exit 1)
	$(GO) install github.com/mfridman/tparse@latest
	@kubectl version --client || (echo "Please install kubectl: https://kubernetes.io/docs/tasks/tools/" && exit 1)
	@docker version --format 'Docker v{{.Server.Version}}' || (echo "Please install Docker Engine: https://docs.docker.com/engine/install" && exit 1)
	@kind --version || (echo "Please install kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation" && exit 1)
	@python3 --version || (echo "Please install python3 https://www.python.org/downloads/ "&& exit 1)

.PHONY: precommit format lint
precommit: format lint
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

#------------------------------------------------------
# Build targets
#------------------------------------------------------
GO ?= CGO_ENABLED=0 go
# Allow setting of go build flags from the command line.
GOFLAGS := 

build:
	@echo "Start go build phase"
	$(GO) build -o ./bin/gwctl ./cmd/gwctl/main.go
	$(GO) build -o ./bin/controlplane ./cmd/controlplane/main.go
	$(GO) build -o ./bin/dataplane ./cmd/dataplane/main.go
	$(GO) build -o ./bin/cl-controlplane ./cmd/cl-controlplane
	$(GO) build -o ./bin/cl-dataplane ./cmd/cl-dataplane
	$(GO) build -o ./bin/cl-go-dataplane ./cmd/cl-go-dataplane
	$(GO) build -o ./bin/cl-adm ./cmd/cl-adm


docker-build: build
	docker build --progress=plain --rm --tag mbg .
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
	cp ./bin/dataplane /usr/local/bin/

clean-tests:
	kind delete cluster --name=mbg1
	kind delete cluster --name=mbg2

#------------------------------------------------------
# Run Targets
#------------------------------------------------------
unit-tests:
	@echo "Running unit tests..."
	$(GO) test -v -count=1 ./pkg/...  -json -cover | tparse --all

tests-e2e: clean-tests 	docker-build 
	$(GO) test -p 1 -timeout 30m -v -tags e2e ./tests/e2e/connectivity/...

tests-iperf3: clean-tests 	docker-build 
	$(GO) test -p 1 -timeout 30m -v -tags e2e ./tests/e2e/iperf3/...

run-gwctl:
	@./bin/gwctl

run-controlplane:
	@./bin/controlplane

run-dataplane:
	@./bin/dataplane

run-kind-iperf3:
	python3 demos/iperf3/kind/simple_test.py -d mtls

run-kind-bookinfo:
	python3 demos/bookinfo/kind/test.py -d mtls

#------------------------------------------------------
# Clean targets
#------------------------------------------------------
clean-kind-iperf3:
	kind delete cluster --name=mbg1
	kind delete cluster --name=mbg2
	kind delete cluster --name=mbg3
	kind delete cluster --name=host-cluster
	kind delete cluster --name=dest-cluster

clean-kind-bookinfo:
	kind delete cluster --name=mbg1
	kind delete cluster --name=mbg2
	kind delete cluster --name=mbg3
	kind delete cluster --name=product-cluster
	kind delete cluster --name=review-cluster

clean-kind: clean-kind-iperf3 clean-kind-bookinfo
