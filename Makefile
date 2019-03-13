OS ?= darwin
ARCH ?= amd64
PREFIX ?= ./build
VERSION := 1.11

.PHONY: install

make:
	mkdir -p ./build
	rm -rf ./build/*
	docker run \
		--rm \
		-e GOOS=$(OS) \
		-e GOARCH=$(ARCH) \
		-v `pwd`:/opt/bin \
		-w /opt/bin golang:$(VERSION) \
			go build -v -o ./build/logger-$(OS)-$(ARCH) ./cmd/logger.go; \

install:
	mv ./build/logger-$(OS)-$(ARCH) $(PREFIX)/logger

test:
	go build -v -o ./build/logger-$(OS)-$(ARCH) ./cmd/logger.go

clean:
	rm -rf ./build
