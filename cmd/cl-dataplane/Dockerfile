FROM envoyproxy/envoy:v1.31.0

# Populated during the build process, for example, with 'arm64' or 'amd64'.
ARG TARGETARCH

# Copy binary
RUN mkdir -p /usr/local/bin
COPY ./bin/$TARGETARCH/cl-dataplane /usr/local/bin/cl-dataplane

# Create directory for private keys
RUN mkdir -p /etc/ssl/private

# Create directory for certificates
RUN mkdir -p /etc/ssl/certs

ENTRYPOINT ["/usr/local/bin/cl-dataplane"]
