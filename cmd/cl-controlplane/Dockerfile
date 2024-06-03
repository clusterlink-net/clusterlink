FROM alpine:3.20

# Populated during the build process, for example, with 'arm64' or 'amd64'.
ARG TARGETARCH

# Copy binary
RUN mkdir -p /usr/local/bin
COPY ./bin/$TARGETARCH/cl-controlplane /usr/local/bin/cl-controlplane

# Create directory for private keys
RUN mkdir -p /etc/ssl/private

# Create directory for certificates
RUN mkdir -p /etc/ssl/certs

# Create directory for store file
RUN mkdir -p /var/lib/clink

ENTRYPOINT ["/usr/local/bin/cl-controlplane"]
