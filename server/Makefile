# eventsgateway
# https://github.com/topfreegames/eventsgateway
#
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2018 Top Free Games <backend@tfgco.com>

GOBIN="${GOPATH}/bin"
TEST_PACKAGES=`find . -type f -name "*.go" ! \( -path "*vendor*" \) | sed -En "s/([^\.])\/.*/\1/p" | uniq`

run:
	@docker compose up eventsgateway-api --wait

build-dev:
	@docker build -t eventsgateway-server-dev -f dev.Dockerfile .

deps-start:
	@docker compose -f ../docker-compose.yaml up -d \
		zookeeper kafka jaeger

deps-stop:
	@docker compose -f ../docker-compose.yaml down

test:
	@docker run -i -v ./:/app eventsgateway-server-dev sh -c 'make test-go'

test-ci:
	@docker run -i eventsgateway-server-dev sh -c 'make test-go'

test-go:
	@${GOBIN}/ginkgo -tags unit -cover -r --randomize-all --randomize-suites ${TEST_PACKAGES}

build-go:
	@mkdir -p bin && go build -o ./bin/eventsgateway main.go

build-image:
	@docker build -t eventsgateway-server -f Dockerfile .