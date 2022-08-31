FROM golang:1.17

# Create dockerfile with multi-stagets: stage 0: compile src and client
# Set destination for COPY
WORKDIR /servicenode

# Download Go modules
COPY go.mod .
RUN go mod download

# Copy the source code.
COPY . ./

# Build Go model
RUN CGO_ENABLED=0 go build -o ./bin/sn ./cmd/servicenode/main.go
RUN CGO_ENABLED=0 go build -o ./bin/server ./cmd/server/main.go


# Create dockerfile with multi-stagets :stage 1: low resources

FROM alpine:3.14

WORKDIR /
COPY --from=0  /servicenode/bin/sn /sn
COPY --from=0  /servicenode/bin/server /server
