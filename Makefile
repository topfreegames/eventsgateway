# eventsgateway
# https://github.com/topfreegames/eventsgateway
#
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2018 Top Free Games <backend@tfgco.com>

MY_IP=`ifconfig | grep --color=none -Eo 'inet (addr:)?([0-9]*\.){3}[0-9]*' | grep --color=none -Eo '([0-9]*\.){3}[0-9]*' | grep -v '127.0.0.1' | head -n 1`
TEST_PACKAGES=`find . -type f -name "*.go" ! \( -path "*server*" \) | sed -En "s/([^\.])\/.*/\1/p" | uniq`
GOBIN="${GOPATH}/bin"

.PHONY: load-test producer spark-notebook

build-dev:
	@docker build -t eventsgateway-client-dev -f dev.Dockerfile .

test:
	docker compose up client-tests

spark-notebook:
	@docker compose up jupyter

producer:
	@echo "Will connect to server at ${MY_IP}:5000"
	@go run main.go producer -d

load-test:
	@echo "Will connect to server at ${MY_IP}:5000"
	@go run main.go load-test -d

deps-start:
	@docker compose up -d eventsgateway-api

setup:
	@go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.4
	@go install github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad
	@go mod tidy
	@cd .git/hooks && ln -sf ./hooks/pre-commit.sh pre-commit

setup-ci:
	@go install github.com/mattn/goveralls@v0.0.11
	@go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.4
	@go install github.com/wadey/gocovmerge@v0.0.0-20160331181800-b5bfa59ec0ad
	@go mod tidy

test-go: unit integration test-coverage-func

unit: print-unit-section unit-run copy-unit-cover

unit-run:
	@${GOBIN}/ginkgo -tags unit -v -cover --covermode count -r --randomize-all --randomize-suites ${TEST_PACKAGES}

integration: print-integration-section integration-run copy-integration-cover

integration-run:
	@GRPC_GO_RETRY=on ${GOBIN}/ginkgo -v -tags integration -cover --covermode count -r --randomize-all --randomize-suites \
		--skip-package=./app ${TEST_PACKAGES}

copy-unit-cover:
	@mkdir -p _build
	@cp ./coverprofile.out _build/coverage-unit.out

copy-integration-cover:
	@mkdir -p _build
	@cp ./coverprofile.out _build/coverage-integration.out

merge-profiles:
	@mkdir -p _build
	@${GOBIN}/gocovmerge _build/*.out > _build/coverage-all.out

print-unit-section:
	@echo
	@echo "=-=-=-=-=-=-=-=-=-=-=-=-=-=-="
	@echo "=        Unit Tests         ="
	@echo "=-=-=-=-=-=-=-=-=-=-=-=-=-=-="

print-integration-section:
	@echo
	@echo "=-=-=-=-=-=-=-=-=-=-=-=-=-=-="
	@echo "=     Integration Tests     ="
	@echo "=-=-=-=-=-=-=-=-=-=-=-=-=-=-="

test-coverage-func: merge-profiles
	@echo
	@echo "=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-="
	@echo "Functions NOT COVERED by Tests  ="
	@echo "=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-="
	@go tool cover -func=_build/coverage-all.out | egrep -v "100.0[%]"

test-coverage-html cover:
	@go tool cover -html=_build/coverage-all.out
