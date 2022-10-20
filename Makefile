#-----------------------------------------------------------------------------
# Target: clean
#-----------------------------------------------------------------------------
.PHONY: clean
clean: ; $(info $(M) cleaning...)	@
	@rm -rf ./bin

#-----------------------------------------------------------------------------
# Target: precommit
#-----------------------------------------------------------------------------
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


build:
	@echo "Start go build phase"
	go build -o ./bin/gateway ./cmd/gateway/...
	go build -o ./bin/mbg ./cmd/mbg/main.go

docker-build-mbg:
	docker build --progress=plain --rm --tag mbg .
docker-build-tcp-split:
	cd manifests/tcp-split/; docker build --progress=plain --rm --tag tcp-split .

docker-build: docker-build-mbg docker-build-tcp-split
run-gateway:
	@./bin/gateway

run-mbg:
	@./bin/mbg

run-kind-mbg:
	kind create cluster --config manifests/kind/config.yaml --name=mbg-agent
	kind load docker-image mbg --name=mbg-agent
	kind load docker-image tcp-split --name=mbg-agent
	kubectl create -f manifests/mbg/mbg.yaml
	kubectl create -f manifests/mbg/mbg-svc.yaml
	kubectl create -f manifests/mbg/mbg-client-svc.yaml
	kubectl create -f manifests/tcp-split/tcp-split.yaml
	kubectl create -f manifests/tcp-split/tcp-split-svc.yaml

run-kind-host:
	kind create cluster --config manifests/kind/config.yaml --name=mbg-agent
	kind load docker-image mbg --name=mbg-agent
	kubectl create -f manifests/host/iperf3-client.yaml
	kubectl create -f manifests/host/gateway-configmap.yaml
	kubectl create -f manifests/host/gateway.yaml
	kubectl create -f manifests/host/gateway-svc.yaml

clean-kind-mbg:
	kind delete cluster --name=mbg-agent

