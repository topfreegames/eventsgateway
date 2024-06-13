FROM golang:1.17-alpine

WORKDIR /app

RUN apk add make build-base

COPY ./go.mod /app/go.mod

RUN go install github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad && \
    go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.4 && \
    go mod tidy
