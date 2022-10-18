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
	go build -o ./bin/client ./cmd/client/...
	go build -o ./bin/server ./cmd/server/main.go
	go build -o ./bin/sn ./cmd/servicenode/main.go
docker-build-sn:
	docker build --progress=plain --rm --tag servicenode .
docker-build-tcp-split:
	cd manifests/tcp-split/; docker build --progress=plain --rm --tag tcp-split .
run-client:
	@./bin/client

run-server:
	@./bin/client
run-sn:
	@./bin/client

run-kind-sn:
	kind create cluster --config manifests/kind/config.yaml --name=ei-agent
	kind load docker-image servicenode --name=ei-agent
	kind load docker-image my-haproxy --name=ei-agent
	kubectl create -f manifests/servidenode/servicenode.yaml
	kubectl create -f manifests/servidenode/sn-svc.yaml
	kubectl create -f manifests/servidenode/sn-client-svc.yaml
	kubectl create -f manifests/tcp-split/haproxy.yaml
	kubectl create -f manifests/tcp-split/split-svc.yaml

run-kind-host:
	kind create cluster --config manifests/kind/config.yaml --name=ei-agent
	kind load docker-image servicenode --name=ei-agent
	kubectl create -f manifests/host/iperf3-client.yaml
	kubectl create -f manifests/host/client-configmap.yaml
	kubectl create -f manifests/host/client.yaml
	kubectl create -f manifests/host/client-svc.yaml

clean-kind-sn:
	kind delete cluster --name=ei-agent

