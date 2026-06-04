.PHONY: all build clean cli import tui

BUILD_DIR := build

all: build

build: cli import tui

cli:
	cd tools && go build -o ../$(BUILD_DIR)/tapass-cli ./cmd/tapass-cli

import:
	cd tools && go build -o ../$(BUILD_DIR)/tapass-import ./cmd/tapass-import

tui:
	cd tui && go build -o ../$(BUILD_DIR)/tapass-tui ./cmd/tapass-tui

clean:
	rm -rf $(BUILD_DIR)
