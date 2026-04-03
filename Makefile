GO ?= go
BUILD_DIR ?= build
BINARY ?= runpost
CMD_PKG ?= ./cmd/runpost

.PHONY: build test clean run

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY) $(CMD_PKG)

test:
	$(GO) test ./...

clean:
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY) --help
