# Can have a different base image from the server.
FROM golang:1.20-alpine

WORKDIR /app

ADD .. /app

RUN apk add make build-base

RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.20.0 && \
    go mod tidy
