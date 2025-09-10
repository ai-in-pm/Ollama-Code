# Simple Makefile for Ollama Code

APP_NAME := ollama-code
BUILD_DIR := dist
BIN := $(BUILD_DIR)/$(APP_NAME)

.PHONY: all build install uninstall clean tidy

all: build

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

build: $(BUILD_DIR)
	GO111MODULE=on go build -o $(BIN)
	@echo "Built $(BIN)"

install: build
	@echo "Installing to /usr/local/bin (requires sudo)"
	sudo install -m 0755 $(BIN) /usr/local/bin/$(APP_NAME)
	sudo ln -sf /usr/local/bin/$(APP_NAME) /usr/bin/olc
	@echo "Installed: /usr/local/bin/$(APP_NAME) and symlink /usr/bin/olc"

uninstall:
	@echo "Removing installed binary and symlink (requires sudo)"
	sudo rm -f /usr/local/bin/$(APP_NAME)
	sudo rm -f /usr/bin/olc
	@echo "Uninstalled"

clean:
	rm -rf $(BUILD_DIR)

# Keep go.mod/go.sum in sync
tidy:
	go mod tidy

