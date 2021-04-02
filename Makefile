GO := go

MAIN_FILE := main.go

BUILD_DIR := build
EXECUTABLE := rmqhttp
BIN_NAME := $(BUILD_DIR)/$(EXECUTABLE)
INSTALLED_NAME := /usr/local/bin/$(EXECUTABLE)

BASE_CMD_DIR := cmd/rmqhttp
BASE_PKG_DIR := pkg

CMD_PACKAGE_DIR := $(BASE_CMD_DIR) $(dir $(wildcard $(BASE_CMD_DIR)/*/))
PKG_PACKAGE_DIR := $(dir $(wildcard $(BASE_PKG_DIR)))
PACKAGE_PATHS := $(CMD_PACKAGE_DIR) $(PKG_PACKAGE_DIR)

AUTOGEN_VERSION_FILENAME=$(BASE_CMD_DIR)/version-temp.go

ALL_GO_DIRS = $(shell find . -iname "*.go" -exec dirname {} \; | sort | uniq)
SRC := $(shell find . -iname "*.go" -and -not -name "*_test.go") $(AUTOGEN_VERSION_FILENAME)
SRC_WITH_TESTS := $(shell find . -iname "*.go") $(AUTOGEN_VERSION_FILENAME)
PUBLISH = publish/linux-amd64 publish/darwin-amd64

.PHONY: all
all: $(BIN_NAME)

$(BIN_NAME): $(SRC)
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BIN_NAME) $(MAIN_FILE)


.PHONY: publish
publish: $(PUBLISH)

.PHONY: publish/linux-amd64
publish/linux-amd64:
	# Force build; don't let existing versions interfere.
	rm -f $(BIN_NAME)
	GOOS=linux GOARCH=amd64 $(MAKE) $(BIN_NAME)
	mkdir -p $$(dirname "$@")
	mv $(BIN_NAME) $@

.PHONY: publish/darwin-amd64
publish/darwin-amd64:
	# Force build; don't let existing versions interfere.
	rm -f $(BIN_NAME)
	GOOS=darwin GOARCH=amd64 $(MAKE) $(BIN_NAME)
	mkdir -p $$(dirname "$@")
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

coverage.out: $(SRC_WITH_TESTS)
	$(GO) test -v --coverprofile=coverage.out ./...

.PHONY: coverage
coverage: coverage.out
	$(GO) tool cover -func=coverage.out

.PHONY: pretty-coverage
pretty-coverage: coverage.out
	$(GO) tool cover -html=coverage.out

.INTERMEDIATE: $(AUTOGEN_VERSION_FILENAME)
$(AUTOGEN_VERSION_FILENAME):
	@version="v$$(cat VERSION)" && \
	build="$$(if [ "$$(git describe)" != "$$version" ]; then echo "-$$(git rev-parse --short HEAD)"; fi)" && \
	dirty="$$(if [ ! -z "$$(git diff; git diff --cached)" ]; then echo "-dirty"; fi)" && \
	printf "package rmqhttp\n\nconst VersionBuild = \"%s%s%s\"" $$version $$build $$dirty > $@

.PHONY: fmt
fmt:
	@$(GO) fmt $(ALL_GO_DIRS)

.PHONY: clean
clean:
	rm -rf coverage.out $(BUILD_DIR)
