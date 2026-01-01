FROM alpine:3.23

# Populated during the build process, for example, with 'arm64' or 'amd64'.
ARG TARGETARCH

# Copy binary
RUN mkdir -p /usr/local/bin
COPY ./bin/$TARGETARCH/cl-go-dataplane /usr/local/bin/cl-go-dataplane

# Create directory for private keys
RUN mkdir -p /etc/ssl/private

# Create directory for certificates
RUN mkdir -p /etc/ssl/certs

ENTRYPOINT ["/usr/local/bin/cl-go-dataplane"]
