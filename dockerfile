FROM golang:1.19

# Create dockerfile with multi-stagets: stage 0: compile src and client
# Set destination for COPY
WORKDIR /gw

# Download Go modules
COPY go.mod .
RUN go mod download

# Copy the source code.
COPY . ./

# Build Go model
RUN CGO_ENABLED=0 go build -o ./bin/controlplane ./cmd/controlplane/main.go
RUN CGO_ENABLED=0 go build -o ./bin/dataplane ./cmd/dataplane/main.go
RUN CGO_ENABLED=0 go build -o ./bin/gwctl ./cmd/gwctl/main.go

# Create dockerfile with multi-stagets :stage 1: low resources

FROM alpine:3.14

WORKDIR /
COPY --from=0  /gw/bin/controlplane /controlplane
COPY --from=0  /gw/bin/dataplane /dataplane
COPY --from=0  /gw/bin/gwctl /gwctl
COPY ./demos/utils/mtls /mtls
# Create the .mbg folder
RUN mkdir -p /root/.gw/
RUN apk update && apk add --no-cache iputils curl tcpdump busybox-extras
