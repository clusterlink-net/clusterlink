FROM envoyproxy/envoy:v1.29.0

# Copy binary
RUN mkdir -p /usr/local/bin
COPY ./bin/cl-dataplane /usr/local/bin/cl-dataplane

# Create directory for private keys
RUN mkdir -p /etc/ssl/private

# Create directory for certificates
RUN mkdir -p /etc/ssl/certs

ENTRYPOINT ["/usr/local/bin/cl-dataplane"]
