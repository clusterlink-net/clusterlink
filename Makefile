#-----------------------------------------------------------------------------
# Target: clean
#-----------------------------------------------------------------------------
.PHONY: clean
clean: ; $(info $(M) cleaning...)	@
	@rm -rf ./bin

#------------------------------------------------------
# Setup Targets
#------------------------------------------------------

prereqs: 											## Verify that required utilities are installed
	@echo -- $@ for MBG Project--
	@go version || (echo "Please install GOLANG: https://go.dev/doc/install" && exit 1)
#	@which goimports || (echo "Please install goimports: https://pkg.go.dev/golang.org/x/tools/cmd/goimports" && exit 1)
	@kubectl version --client || (echo "Please install kubectl: https://kubernetes.io/docs/tasks/tools/" && exit 1)
	@docker version --format 'Docker v{{.Server.Version}}' || (echo "Please install Docker Engine: https://docs.docker.com/engine/install" && exit 1)
	@kind --version || (echo "Please install kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation" && exit 1)
	@python3 --version || (echo "Please install python3 https://www.python.org/downloads/ "&& exit 1)

.PHONY: precommit format lint
precommit: format lint
format: fmt
fmt: format-go tidy-go vet-go
vet: vet-go

tidy-go: ; $(info $(M) tidying up go.mod...)
	@go mod tidy

format-go: tidy-go vet-go ; $(info $(M) formatting code...)
	@go fmt ./...

vet-go: ; $(info $(M) vetting code...)
	@go vet ./...

#------------------------------------------------------
# Build targets
#------------------------------------------------------


build:
	@echo "Start go build phase"
	go build -o ./bin/mbgctl ./cmd/mbgctl/main.go
	go build -o ./bin/mbg ./cmd/mbg/main.go

docker-build-mbg:
	docker build --progress=plain --rm --tag mbg .

docker-build: docker-build-mbg 

proto-build:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/protocol/protocol.proto

install:
	cp ./bin/mbgctl /usr/local/bin/
	cp ./bin/mbg /usr/local/bin/
#------------------------------------------------------
# Run Targets
#------------------------------------------------------
run-mbgctl:
	@./bin/mbgctl

run-mbg:
	@./bin/mbg

run-kind-iperf3:
	python3 tests/iperf3/kind/allinone.py -d mtls

run-kind-bookinfo:
	python3 tests/bookinfo/kind/test.py -d mtls

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