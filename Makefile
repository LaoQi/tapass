.PHONY: all build clean cli import tui dist package

BUILD_DIR := build
DIST_DIR  := dist

GOOS   ?=
GOARCH ?=

EXE_SUFFIX :=
ifeq ($(GOOS),windows)
  EXE_SUFFIX := .exe
endif

VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo "")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "")
ARCHIVE_VERSION := $(if $(VERSION),$(VERSION),$(if $(COMMIT),$(COMMIT),dev))
TARGET  := $(if $(GOOS),$(GOOS)-$(GOARCH),$(shell go env GOOS)-$(shell go env GOARCH))
ARCHIVE := $(DIST_DIR)/tapass-$(ARCHIVE_VERSION)-$(TARGET).zip

LDFLAGS := -X github.com/LaoQi/tapass/tools/version.Version=$(VERSION) \
           -X github.com/LaoQi/tapass/tools/version.Commit=$(COMMIT)

all: build

build: cli import tui

cli:
	cd tools && CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o ../$(BUILD_DIR)/tapass-cli$(EXE_SUFFIX) ./cmd/tapass-cli

import:
	cd tools && CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o ../$(BUILD_DIR)/tapass-import$(EXE_SUFFIX) ./cmd/tapass-import

tui:
	cd tui && CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "$(LDFLAGS)" -o ../$(BUILD_DIR)/tapass-tui$(EXE_SUFFIX) ./cmd/tapass-tui

dist: build
	@mkdir -p $(DIST_DIR)
	cp CHANGELOG.md $(BUILD_DIR)/
	cd $(BUILD_DIR) && zip -j ../$(ARCHIVE) tapass-cli$(EXE_SUFFIX) tapass-import$(EXE_SUFFIX) tapass-tui$(EXE_SUFFIX) CHANGELOG.md

package: dist

clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)
