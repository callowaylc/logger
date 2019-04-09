#!make
OS ?= darwin
ARCH ?= amd64
PREFIX ?= ./build
VERSION := 1.11
TAG = $(shell date -u +'%Y.%m.%d-%H')
ORG = callowaylc
REPO = logger
ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))

-include .env
export

.PHONY: make install release test clean

make:
	mkdir -p ./build
	docker run \
		--rm \
		-e GOOS=$(OS) \
		-e GOARCH=$(ARCH) \
		-v `pwd`:/opt/bin \
		-v `pwd`/build/cache:/go/pkg \
		-w /opt/bin golang:$(VERSION) \
			go build -v -o ./release/logger-$(OS)-$(ARCH) ./cmd/logger.go; \

install:
	mv ./build/logger-$(OS)-$(ARCH) $(PREFIX)/logger

release:
	# NOTE: Add latest along with calendar version
	mkdir -p ./release
	rm -rf ./release/*

	OS=darwin make & \
	OS=linux make & \
	wait

	git tag $(TAG) -f
	git push origin $(TAG) -f

	- github-release delete \
		--user $(ORG) \
		--repo $(REPO) \
		--tag $(TAG)

	github-release release --draft \
		--user $(ORG) \
		--repo $(REPO) \
		--tag $(TAG) \
		--name $(TAG)

	ls ./release/* | xargs -n1 basename | xargs -n1 -I{} github-release upload \
		--replace \
		--user $(ORG) \
		--repo $(REPO) \
		--tag $(TAG) \
		--name {} \
    --file ./release/{}

publish:
	github-release edit \
		--user $(ORG) \
		--repo $(REPO) \
		--tag $(TAG) \
		--name $(TAG)

test:
	vgo build -v -o ./build/logger-$(OS)-$(ARCH) ./cmd/logger.go

clean:
	rm -rf ./build

%:
	@:
