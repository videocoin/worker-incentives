JOB_IMAGE = registry.videocoin.net/vworker-incentives/server
VERSION ?= $(shell git describe --tags)

.PHONY: build
build:
	go build -o ./build/server ./cmd

.PHONY: deps
deps:
	go get github.com/goware/modvendor

.PHONY: vendor
vendor:
	go mod vendor
	modvendor -copy="**/*.c **/*.h"

.PHONY: test
test:
	go test ./...

.PHONY: image
image:
	docker build -t ${JOB_IMAGE}:$(VERSION) -f _assets/Dockerfile .
	docker tag ${JOB_IMAGE}:$(VERSION) ${JOB_IMAGE}:latest

.PHONY: push
push:
	docker push ${JOB_IMAGE}:${VERSION}
	docker push ${JOB_IMAGE}:latest

.PHONY: release
release: image push
