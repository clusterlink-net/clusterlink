FROM golang:1.19

# Create dockerfile with multi-stagets: stage 0: compile src and client
# Set destination for COPY
WORKDIR /mbg

# Download Go modules
COPY go.mod .
RUN go mod download

# Copy the source code.
COPY . ./

# Build Go model
RUN CGO_ENABLED=0 go build -o ./bin/mbg ./cmd/mbg/main.go
RUN CGO_ENABLED=0 go build -o ./bin/mbgctl ./cmd/mbgctl/main.go

# Create dockerfile with multi-stagets :stage 1: low resources

FROM alpine:3.14

WORKDIR /
COPY --from=0  /mbg/bin/mbg /mbg
COPY --from=0  /mbg/bin/mbgctl /mbgctl
COPY ./tests/utils/mtls /mtls
# Create the .mbg folder
RUN mkdir -p /root/.mbg/
RUN apk update && apk add --no-cache iputils curl tcpdump busybox-extras
