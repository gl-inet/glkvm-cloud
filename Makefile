# Makefile
# ---------------- Project ----------------
BINARY_NAME ?= rttys
UI_DIR      ?= ui
CONF_FILE   ?= ./rttys.conf

# Go build flags
BUILD_FLAGS ?= -ldflags "-s -w"

# Output dir for cross builds
DIST_DIR    ?= dist

# Image name
IMAGE_NAME  ?= glkvm-cloud
IMAGE_TAG   ?= build

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Map uname -m -> goarch
ifeq ($(UNAME_M),x86_64)
  HOST_GOARCH := amd64
else ifeq ($(UNAME_M),aarch64)
  HOST_GOARCH := arm64
else ifeq ($(UNAME_M),arm64)
  HOST_GOARCH := arm64
else
  HOST_GOARCH := $(GOARCH)
endif

# ---------------- Commands ----------------
GO_BUILD_CMD = go build $(BUILD_FLAGS) -o $(BINARY_NAME)

.PHONY: all ui build run build-all build-run full-run \
        build-linux-amd64 build-linux-arm64 build-linux-all \
        docker-build docker-fullbuild docker-buildx docker-buildx-full

all: build

# Build frontend files only
ui:
	cd $(UI_DIR) && npm install && npm run build

# Build for current env (native)
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO_BUILD_CMD)

# Run Go program only (native binary)
run:
	./$(BINARY_NAME) -c $(CONF_FILE)

# Build frontend and Go binary
build-all: ui build

# Build Go binary and run
build-run: build run

# Build frontend, build Go binary, and run
full-run: ui build run

# ---------------- Cross compile (Linux) ----------------
# Produce: dist/rttys-linux-amd64 , dist/rttys-linux-arm64
build-linux-amd64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64

build-linux-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64

build-linux-all: build-linux-amd64 build-linux-arm64

# ---------------- Docker (single-arch) ----------------
# Build Docker image using current host arch
docker-build: build
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# Full build Docker image
docker-fullbuild: ui build
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

# ---------------- Docker Buildx ----------------
# Multi-arch build
# Usage:
#   make docker-buildx GOARCH=amd64 IMAGE_TAG=build-amd64
#   make docker-buildx GOARCH=arm64 IMAGE_TAG=build-arm64
PLATFORMS ?= linux/amd64,linux/arm64
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
