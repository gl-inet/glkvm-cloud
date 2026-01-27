# Makefile
# ---------------- Project ----------------
BINARY_NAME ?= rttys
UI_DIR      ?= ui
GO_MAIN     ?= ./cmd/glkvm-cloud

# Go build flags
BUILD_FLAGS ?= -ldflags "-s -w"
DIST_DIR    ?= dist

# Image name
IMAGE_NAME  ?= glkvm-cloud
IMAGE_TAG   ?= build

GOARCH ?= $(shell go env GOARCH)

# ---------------- Commands ----------------
.PHONY: all ui debug-local \
        build-linux-amd64 build-linux-arm64 build-linux-all \
        docker-buildx docker-buildx-full

all: build-linux-amd64 build-linux-arm64

# Build frontend files only
ui:
	cd $(UI_DIR) && npm install && npm run build

# ---------------- Cross compile (Linux) ----------------
# Produce: dist/rttys-linux-amd64 , dist/rttys-linux-arm64
build-linux-amd64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(GO_MAIN)

build-linux-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(GO_MAIN)

# ---------------- Docker Buildx ----------------
# Multi-arch build
# Usage:
#   make docker-buildx GOARCH=amd64 IMAGE_TAG=build-amd64
#   make docker-buildx GOARCH=arm64 IMAGE_TAG=build-arm64
REGISTRY  ?=

# If REGISTRY is set, tag becomes: REGISTRY/IMAGE_NAME:IMAGE_TAG
ifdef REGISTRY
  IMAGE_REF := $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
else
  IMAGE_REF := $(IMAGE_NAME):$(IMAGE_TAG)
endif

docker-buildx:
	@docker buildx version >/dev/null 2>&1 || (echo "docker buildx not available" && exit 1)
	@echo "==> buildx (load local image): $(IMAGE_REF) [linux/$(GOARCH)]"
	docker buildx build \
		--platform linux/$(GOARCH) \
		-t $(IMAGE_REF) \
		--load .

docker-buildx-full: ui
	@$(MAKE) docker-buildx


DEBUG_HOST  ?= root@107.173.152.173
DEBUG_PATH  ?= /root/glkvmcloudbuild.tar
# Local debug bundle (amd64 image + save tar), upload to debug host, then load
debug-local: build-linux-amd64 docker-buildx
	docker save $(IMAGE_NAME):$(IMAGE_TAG) -o glkvmcloudbuild.tar
	ssh $(DEBUG_HOST) "rm -f $(DEBUG_PATH)"
	scp glkvmcloudbuild.tar $(DEBUG_HOST):$(DEBUG_PATH)
	ssh $(DEBUG_HOST) "docker load < $(DEBUG_PATH)"