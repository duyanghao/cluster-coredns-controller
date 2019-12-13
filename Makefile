## Server version
SERVER_VERSION = v0.1.0
## Folder content generated files
BUILD_FOLDER = ./build
PROJECT_URL  = github.com/duyanghao/coredns-sync
## command
GO           = go
GO_VENDOR    = go mod
MKDIR_P      = mkdir -p

## Random Alphanumeric String
SECRET_KEY   = $(shell cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)

## UNAME
UNAME := $(shell uname)

################################################

.PHONY: all
all: build

.PHONY: pre-build
pre-build:
	$(GO_VENDOR) download

.PHONY: build
build: pre-build
	$(MAKE) src.build

## vendor/ #####################################

.PHONY: download
download:
	$(GO_VENDOR) download

## src/ ########################################

.PHONY: src.build
src.build:
	cd cmd && GO111MODULE=on $(GO) build -mod=vendor -v -o ../$(BUILD_FOLDER)/coredns-sync/coredns-sync

## dockerfiles/ ########################################

.PHONY: dockerfiles.build
dockerfiles.build:
	docker build --tag duyanghao/coredns-sync:$(SERVER_VERSION) -f ./docker/Dockerfile .

## git tag version ########################################

.PHONY: push.tag
push.tag:
	@echo "Current git tag version:"$(SERVER_VERSION)
	git tag $(SERVER_VERSION)
	git push --tags
