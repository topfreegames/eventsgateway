# Can have a different base image from the server.
FROM golang:1.24-alpine

WORKDIR /app

ADD .. /app

RUN apk add make build-base

RUN go install github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad && \
    go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.4 && \
    go mod tidy
