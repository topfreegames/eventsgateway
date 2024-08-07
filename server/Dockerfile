#
# Copyright (c) 2018 TFG Co <backend@tfgco.com>
# Author: TFG Co <backend@tfgco.com>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy of
# this software and associated documentation files (the "Software"), to deal in
# the Software without restriction, including without limitation the rights to
# use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
# the Software, and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
# FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
# COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
# IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
#

FROM golang:1.22-alpine AS build-env

MAINTAINER TFG Co <backend@tfgco.com>

RUN mkdir -p /app/bin

ADD .. /go/src/github.com/topfreegames/eventsgateway

RUN apk add --no-cache make git g++ && \
    cd /go/src/github.com/topfreegames/eventsgateway && \
    make build-go && \
    mv bin/eventsgateway /app/eventsgateway && \
    mv config /app/config

FROM alpine:3.9

WORKDIR /app

COPY --from=build-env /app/eventsgateway /app/eventsgateway
COPY --from=build-env /app/config /app/config

EXPOSE 8080

ENV EVENTSGATEWAY_KAFKA_PRODUCER_BROKERS kafka:9092
ENV EVENTSGATEWAY_KAFKA_PRODUCER_MAXMESSAGEBYTES 3000000

CMD /app/eventsgateway start -j
