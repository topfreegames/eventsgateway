# eventsgateway
# https://github.com/topfreegames/eventsgateway
#
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2018 Top Free Games <backend@tfgco.com>

MY_IP=`ifconfig | grep --color=none -Eo 'inet (addr:)?([0-9]*\.){3}[0-9]*' | grep --color=none -Eo '([0-9]*\.){3}[0-9]*' | grep -v '127.0.0.1' | head -n 1`
TEST_PACKAGES=`find . -type f -name "*.go" ! \( -path "*vendor*" \) | sed -En "s/([^\.])\/.*/\1/p" | uniq`

.PHONY: testclient gobblin

setup: setup-hooks
	@go get -u github.com/golang/dep...
	@go get -u github.com/wadey/gocovmerge
	@dep ensure

setup-hooks:
	@cd .git/hooks && ln -sf ./hooks/pre-commit.sh pre-commit

setup-ci:
	@go get github.com/mattn/goveralls
	@go get -u github.com/golang/dep/cmd/dep
	@go get github.com/onsi/ginkgo/ginkgo
	@go get -u github.com/wadey/gocovmerge
	@dep ensure

build:
	@mkdir -p bin && go build -o ./bin/eventsgateway main.go

build-docker:
	@docker build -t eventsgateway .

deps-start:
	@echo "Starting dependencies using HOST IP of ${MY_IP}..."
	@env MY_IP=${MY_IP} docker-compose --project-name eventsgateway up -d \
		zookeeper kafka localstack 
	@echo "Dependencies started successfully."

deps-stop:
	@env MY_IP=${MY_IP} docker-compose --project-name eventsgateway down

cross-build-linux-amd64:
	@env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bin/eventsgateway-linux-amd64
	@chmod a+x ./bin/eventsgateway-linux-amd64

run:
	@echo "Will connect to kafka at ${MY_IP}:9192"
	@env EVENTSGATEWAY_EXTENSIONS_KAFKAPRODUCER_BROKERS=${MY_IP}:9192 go run main.go start -d

testclient:
	@echo "Will connect to server at ${MY_IP}:5000"
	@env EVENTSGATEWAY_PROMETHEUS_PORT=9092 go run main.go testclient -d

hive-start:
	@echo "Starting Hive stack using HOST IP of ${MY_IP}..."
	@cd ./hive && docker-compose up
	@echo "Hive stack started successfully."

hive-stop:
	@cd ./hive && docker-compose down

gobblin:
	@echo "Starting Gobblin using HOST IP of ${MY_IP}..."
	@docker stop eventsgateway_gobblin_1
	@mv ./gobblin/conf/events.pull.done ./gobblin/conf/events.pull
	@rm -rf ./gobblin/work-dir && mkdir ./gobblin/work-dir
	@env MY_IP=${MY_IP} docker-compose up gobblin
	@echo "Gobblin started successfully."

unit: unit-board clear-coverage-profiles unit-run gather-unit-profiles

clear-coverage-profiles:
	@find . -name '*.coverprofile' -delete

unit-board:
	@echo
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"
	@echo "\033[1;34m=         Unit Tests         -\033[0m"
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"

unit-run:
	@ginkgo -tags unit -cover -r -randomizeAllSpecs -randomizeSuites -skipMeasurements ${TEST_PACKAGES}

gather-unit-profiles:
	@mkdir -p _build
	@echo "mode: count" > _build/coverage-unit.out
	@bash -c 'for f in $$(find . -name "*.coverprofile"); do tail -n +2 $$f >> _build/coverage-unit.out; done'

int integration: integration-board clear-coverage-profiles deps-test integration-run gather-integration-profiles

integration-board:
	@echo
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"
	@echo "\033[1;34m=     Integration Tests      -\033[0m"
	@echo "\033[1;34m=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-\033[0m"

integration-run:
	@ginkgo -tags integration -cover -r -randomizeAllSpecs -randomizeSuites -skipMeasurements ${TEST_PACKAGES}

int-ci: integration-board clear-coverage-profiles deps-test-ci integration-run gather-integration-profiles

gather-integration-profiles:
	@mkdir -p _build
	@echo "mode: count" > _build/coverage-integration.out
	@bash -c 'for f in $$(find . -name "*.coverprofile"); do tail -n +2 $$f >> _build/coverage-integration.out; done'

merge-profiles:
	@mkdir -p _build
	@gocovmerge _build/*.out > _build/coverage-all.out

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
