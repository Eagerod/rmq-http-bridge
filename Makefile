GO := go

MAIN_FILE := main.go

BUILD_DIR := build
EXECUTABLE := rmqhttp
BIN_NAME := $(BUILD_DIR)/$(EXECUTABLE)
INSTALLED_NAME := /usr/local/bin/$(EXECUTABLE)

COVERAGE_FILE=./coverage.out

BASE_CMD_DIR := cmd/rmqhttp
BASE_PKG_DIR := pkg

CMD_PACKAGE_DIR := $(BASE_CMD_DIR) $(dir $(wildcard $(BASE_CMD_DIR)/*/))
PKG_PACKAGE_DIR := $(dir $(wildcard $(BASE_PKG_DIR)))
PACKAGE_PATHS := $(CMD_PACKAGE_DIR) $(PKG_PACKAGE_DIR)

SRC := $(shell find . -iname "*.go" -and -not -name "*_test.go")
SRC_WITH_TESTS := $(shell find . -iname "*.go")

PUBLISH_DIR=publish
PUBLISH := \
	$(PUBLISH_DIR)/linux-amd64 \
	$(PUBLISH_DIR)/darwin-amd64 \
	$(PUBLISH_DIR)/darwin-arm64

DOCKER_IMAGE_NAME = rmq-http-bridge

.PHONY: all
all: $(BIN_NAME)

$(BIN_NAME): $(SRC)
	@mkdir -p $(BUILD_DIR)
	version="$${VERSION:-$$(git describe --dirty)}"; \
	$(GO) build -o $(BIN_NAME) -ldflags="-X github.com/Eagerod/rmqhttp/cmd/rmqhttp.VersionBuild=$$version" $(MAIN_FILE)



.PHONY: publish
publish: $(PUBLISH)

# Publish targets are treated as phony to force rebuilds.
.PHONY: $(PUBLISH)
$(PUBLISH):
	mkdir -p "$(@D)"
	rm -f $(BIN_NAME)
	GOOS_GOARCH="$$(basename $@)" \
	GOOS="$$(cut -d '-' -f 1 <<< "$$GOOS_GOARCH")" \
	GOARCH="$$(cut -d '-' -f 2 <<< "$$GOOS_GOARCH")" \
		$(MAKE) $(BIN_NAME)
	mv $(BIN_NAME) $@


.PHONY: server worker
server worker: $(BIN_NAME)
	$(BIN_NAME) $@ --queue test


.PHONY: install isntall
install isntall: $(INSTALLED_NAME)

$(INSTALLED_NAME): $(BIN_NAME)
	cp $(BIN_NAME) $(INSTALLED_NAME)

.PHONY: test
test:
	@$(GO) vet ./...
	@staticcheck ./...
	@if [ -z $$T ]; then \
		$(GO) test -v ./...; \
	else \
		$(GO) test -v ./... -run $$T; \
	fi

.PHONY: interface-test
interface-test: $(BIN_NAME)
	@if [ -z $$T ]; then \
		$(GO) test -v main_test.go; \
	else \
		$(GO) test -v main_test.go -run $$T; \
	fi

$(COVERAGE_FILE): $(SRC_WITH_TESTS)
	$(GO) test -v --coverprofile=$(COVERAGE_FILE) ./...

.PHONY: coverage
coverage: $(COVERAGE_FILE)
	$(GO) tool cover -func=$(COVERAGE_FILE)

.PHONY: pretty-coverage
pretty-coverage: $(COVERAGE_FILE)
	$(GO) tool cover -html=$(COVERAGE_FILE)

.PHONY: fmt
fmt:
	@$(GO) fmt ./...

.PHONY: clean
clean:
	rm -rf $(COVERAGE_FILE) $(BUILD_DIR)


.PHONY: container
container:
	@version="$$(git describe --dirty | sed 's/^v//')"; \
	docker build . --build-arg VERSION="$$version" -t "registry.internal.aleemhaji.com/$(DOCKER_IMAGE_NAME):$$version"
