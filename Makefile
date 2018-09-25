PACKAGE = github.com/starcloud-ai/kubeconfig
COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`
BUILD_IMAGE = golang:1.11-alpine
IMAGE = asdfsx/kubeconfig
WORKDIR = /go/src/${PACKAGE}
TARGET = kubeconfig

.PHONY: all build clean docker push help

clean: ## Delete kubeconfig binary
	-rm ${TARGET}

vendor: ## add dependencies to vendor directory
	go mod vendor

lint: vendor ## use gometalinter to check code
	GO111MODULE=off gometalinter --skip=./vendor --deadline=5m

build: clean vendor ## build for docker
	docker run --rm -v "${PWD}":"${WORKDIR}" -w ${WORKDIR} ${BUILD_IMAGE} go build -o ${TARGET}
	docker build -t ${IMAGE} .

push: ## push image to docker registry
	docker push ${IMAGE}

all: build push ## build binary, then build and push image

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'