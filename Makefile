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
	@echo -- $@ --
	@go version || (echo "Please install GOLANG: https://go.dev/doc/install" && exit 1)
#	@which goimports || (echo "Please install goimports: https://pkg.go.dev/golang.org/x/tools/cmd/goimports" && exit 1)
	@kubectl version --client || (echo "Please install kubectl: https://kubernetes.io/docs/tasks/tools/" && exit 1)
	@docker version --format 'Docker v{{.Server.Version}}' || (echo "Please install Docker Engine: https://docs.docker.com/engine/install" && exit 1)
	@kind --version || (echo "Please install kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation" && exit 1)

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
	go build -o ./bin/cluster ./cmd/cluster/main.go
	go build -o ./bin/mbg ./cmd/mbg/main.go

docker-build-mbg:
	docker build --progress=plain --rm --tag mbg .
docker-build-tcp-split:
	cd manifests/tcp-split/; docker build --progress=plain --rm --tag tcp-split .

docker-build: docker-build-mbg docker-build-tcp-split

proto-build:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/protocol/protocol.proto


#------------------------------------------------------
# Run Targets
#------------------------------------------------------
run-cluster:
	@./bin/cluster

run-mbg:
	@./bin/mbg

run-kind-mbg1:
	kind create cluster --config manifests/kind/mbg-config1.yaml --name=mbg-agent1
	kind load docker-image mbg --name=mbg-agent1
	kind load docker-image tcp-split --name=mbg-agent1
	kubectl create -f manifests/mbg/mbg.yaml
	kubectl create -f manifests/mbg/mbg-svc.yaml
	kubectl create -f manifests/mbg/mbg-client-svc.yaml
	kubectl create -f manifests/tcp-split/tcp-split.yaml
	kubectl create -f manifests/tcp-split/tcp-split-svc.yaml

run-kind-mbg2:
	kind create cluster --config manifests/kind/mbg-config2.yaml --name=mbg-agent2
	kind load docker-image mbg --name=mbg-agent2
	kind load docker-image tcp-split --name=mbg-agent2
	kubectl create -f manifests/mbg/mbg.yaml
	kubectl create -f manifests/mbg/mbg-svc.yaml
	kubectl create -f manifests/mbg/mbg-client-svc.yaml
	kubectl create -f manifests/tcp-split/tcp-split.yaml
	kubectl create -f manifests/tcp-split/tcp-split-svc.yaml

run-kind-host:
	kind create cluster --config manifests/kind/host-config.yaml --name=cluster-host
	kind load docker-image mbg --name=cluster-host
	kubectl create -f manifests/host/iperf3/iperf3-client.yaml
	kubectl create -f manifests/host/iperf3/iperf3-svc.yaml
	kubectl create -f manifests/cluster/cluster.yaml
	kubectl create -f manifests/cluster/cluster-svc.yaml

run-kind-dest:
	kind create cluster --config manifests/kind/dest-config.yaml --name=cluster-dest
	kind load docker-image mbg --name=cluster-dest
	kubectl create -f manifests/dest/iperf3/iperf3.yaml
	kubectl create -f manifests/cluster/cluster.yaml
	kubectl create -f manifests/cluster/cluster-svc.yaml

#------------------------------------------------------
# Clena targets
#------------------------------------------------------
clean-kind-mbg:
	kind delete cluster --name=mbg-agent1
	kind delete cluster --name=mbg-agent2
	kind delete cluster --name=cluster-host
	kind delete cluster --name=cluster-dest
