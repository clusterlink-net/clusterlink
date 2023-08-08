FROM alpine:3.14

# Copy binary
RUN mkdir -p /usr/local/bin
COPY ./bin/gwctl /usr/local/bin/gwctl

# Install bash
RUN apk add bash
