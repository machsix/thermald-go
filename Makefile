# Makefile for building
NAME:=thermald-go
BINARY:=thermald-go
OUTPUT_DIR?=build/linux/$(LINUX_ARCH)
LINUX_ARCH ?= amd64
SYSTEM:= GOOS=linux GOARCH=$(LINUX_ARCH)
BUILDOPTS?=
GOPATH?=$(HOME)/go
CGO_ENABLED?=0
GOLANG_VERSION ?= $(shell grep "^go" go.mod | awk '{print $$2}')
VERSION ?= dev-$(shell git rev-parse --short HEAD)

export GOSUMDB = sum.golang.org
export GOTOOLCHAIN = go$(GOLANG_VERSION)

.PHONY: all
all: thermald-go tar

.PHONY: thermald-go
thermald-go:
	@echo Building: linux/$(LINUX_ARCH) - $(VERSION)
	mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(SYSTEM) go build $(BUILDOPTS) -ldflags="-s -w -X main.Version=$(VERSION)" -o $(OUTPUT_DIR) ./...

.PHONY: tar
tar:
	@echo Packing: linux/$(LINUX_ARCH) - $(VERSION)
	mkdir -p release
	tar -zcf release/$(NAME)_$(VERSION)_linux_$(LINUX_ARCH).tgz -C $(OUTPUT_DIR) $(BINARY)  ;\

.PHONY: clean
clean:
	go clean
	rm -rf $(OUTPUT_DIR)

