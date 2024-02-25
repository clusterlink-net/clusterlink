FROM alpine:3.14

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY  ./bin/cl-operator cl-operator
USER 65532:65532

ENTRYPOINT ["/cl-operator"]
