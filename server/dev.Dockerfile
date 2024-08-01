# Can have a different base image from the client.
FROM golang:1.22-alpine

WORKDIR /app

ADD .. /app

RUN apk add make build-base

RUN go install github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad && \
    go install github.com/onsi/ginkgo/v2/ginkgo@v2.19.1 && \
    go mod tidy
