FROM alpine:3.14

# Copy binary
RUN mkdir -p /usr/local/bin
COPY ./bin/cl-go-dataplane /usr/local/bin/cl-go-dataplane

# Create directory for private keys
RUN mkdir -p /etc/ssl/private

# Create directory for certificates
RUN mkdir -p /etc/ssl/certs

ENTRYPOINT ["/usr/local/bin/cl-go-dataplane"]
