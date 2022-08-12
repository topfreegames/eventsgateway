# eventsgateway
# https://github.com/topfreegames/eventsgateway
#
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2018 Top Free Games <backend@tfgco.com>

MY_IP=`ifconfig | grep --color=none -Eo 'inet (addr:)?([0-9]*\.){3}[0-9]*' | grep --color=none -Eo '([0-9]*\.){3}[0-9]*' | grep -v '127.0.0.1' | head -n 1`
TEST_PACKAGES=`find . -type f -name "*.go" ! \( -path "*vendor*" \) | sed -En "s/([^\.])\/.*/\1/p" | uniq`
GOBIN="${GOPATH}/bin"

.PHONY: load-test producer spark-notebook

setup: setup-hooks
	@go install github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad
	@go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.4
	@go mod tidy

setup-hooks:
	@cd .git/hooks && ln -sf ./hooks/pre-commit.sh pre-commit

setup-ci:
	@go install github.com/mattn/goveralls@v0.0.11
	@go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.4
	@go install github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad
	@go mod tidy

build:
	@mkdir -p bin && go build -o ./bin/eventsgateway main.go

build-docker:
	@docker build -t eventsgateway .

deps-start:
	@echo "Starting dependencies using HOST IP of ${MY_IP}..."
	@env MY_IP=${MY_IP} docker compose --project-name eventsgateway up -d \
		zookeeper kafka localstack jaeger
	@echo "Dependencies started successfully."

deps-stop:
	@env MY_IP=${MY_IP} docker compose --project-name eventsgateway down

deps-test-start:
	@env MY_IP=${MY_IP} docker compose --project-name eventsgateway-test up -d \
		zookeeper kafka

deps-test-stop:
	@env MY_IP=${MY_IP} docker compose --project-name eventsgateway-test down

cross-build-linux-amd64:
	@env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/eventsgateway-linux-amd64
	@chmod a+x ./bin/eventsgateway-linux-amd64

run:
	@echo "Will connect to kafka at ${MY_IP}:9192"
	@echo "OTLP exporter endpoint at http://${MY_IP}:4317"
	@env EVENTSGATEWAY_EXTENSIONS_KAFKAPRODUCER_BROKERS=${MY_IP}:9192 OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://${MY_IP}:4317 go run main.go start -d


producer:
	@echo "Will connect to server at ${MY_IP}:5000"
	@env EVENTSGATEWAY_PROMETHEUS_PORT=9092 go run main.go producer -d

load-test:
	@echo "Will connect to server at ${MY_IP}:5000"
	@env EVENTSGATEWAY_PROMETHEUS_PORT=9092 go run main.go load-test -d

spark-notebook:
	@env MY_IP=${MY_IP} docker compose --project-name eventsgateway up -d \
		spark-notebook

hive-start:
	@echo "Starting Hive stack using HOST IP of ${MY_IP}..."
	@cd ./hive && docker compose up
	@echo "Hive stack started successfully."

hive-stop:
	@cd ./hive && docker compose down

unit: unit-board clear-coverage-profiles unit-run gather-unit-profiles

clear-coverage-profiles:
	@find . -name '*.coverprofile' -delete

unit-board:
	@echo
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"
	@echo "\033[1;34m=         Unit Tests         -\033[0m"
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"

unit-run:
	@${GOBIN}/ginkgo -tags unit -cover -r --randomize-all --randomize-suites ${TEST_PACKAGES}

gather-unit-profiles:
	@mkdir -p _build
	@echo "mode: count" > _build/coverage-unit.out
	@bash -c 'for f in $$(find . -name "*.coverprofile"); do tail -n +2 $$f >> _build/coverage-unit.out; done'

int integration: integration-board clear-coverage-profiles deps-test-start integration-run gather-integration-profiles deps-test-stop

integration-board:
	@echo
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"
	@echo "\033[1;34m=     Integration Tests      -\033[0m"
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"

integration-run:
	@GRPC_GO_RETRY=on ${GOBIN}/ginkgo -tags integration -cover -r --randomize-all --randomize-suites \
		--skip-package=./app ${TEST_PACKAGES}

int-ci: integration-board clear-coverage-profiles deps-test-ci integration-run gather-integration-profiles

gather-integration-profiles:
	@mkdir -p _build
	@echo "mode: count" > _build/coverage-integration.out
	@bash -c 'for f in $$(find . -name "*.coverprofile"); do tail -n +2 $$f >> _build/coverage-integration.out; done'

merge-profiles:
	@mkdir -p _build
	@${GOBIN}/gocovmerge _build/*.out > _build/coverage-all.out

test-coverage-func coverage-func: merge-profiles
	@echo
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"
	@echo "\033[1;34mFunctions NOT COVERED by Tests\033[0m"
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"
	@go tool cover -func=_build/coverage-all.out | egrep -v "100.0[%]"

test: unit int test-coverage-func

test-ci: unit test-coverage-func

test-coverage-html cover:
	@go tool cover -html=_build/coverage-all.out

rtfd:
	@rm -rf docs/_build
	@sphinx-build -b html -d ./docs/_build/doctrees ./docs/ docs/_build/html
	@open docs/_build/html/index.html
