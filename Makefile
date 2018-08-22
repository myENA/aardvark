REGISTRY ?=
IMAGE_PATH ?= $(REGISTRY)ena/aardvark
SUDO ?=
RELEASE_VERSION = $(shell grep RELEASE_VERSION= build/build.sh | grep -oE '[0-9]+?\.[0-9]+?')

ifeq ($(SUDO),true)
	sudo = sudo
endif

.PHONY: build test check clean docker docker-release

build:
	@build/build.sh

test:
	@go test -v

check:
	@build/codeCheck.sh

clean:
	@build/build.sh -d

docker:
	$(sudo) docker build -t $(IMAGE_PATH):latest .

docker-release:
	$(sudo) docker build -t $(IMAGE_PATH):latest .
	$(sudo) docker tag $(IMAGE_PATH):latest $(IMAGE_PATH):$(RELEASE_VERSION)
