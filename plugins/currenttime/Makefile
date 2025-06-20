# CurrentTime Plugin Makefile

PLUGIN_NAME = currenttime
SO_FILE = $(PLUGIN_NAME).so
MAIN_FILE = main.go

# Go build flags for plugin
GO_BUILD_FLAGS = -buildmode=plugin -ldflags="-s -w"

# Default target
all: build

# Build the plugin
build:
	@echo "Building $(PLUGIN_NAME) plugin..."
	go build $(GO_BUILD_FLAGS) -o $(SO_FILE) $(MAIN_FILE)
	@echo "Plugin built successfully: $(SO_FILE)"

# Clean build artifacts
clean:
	@echo "Cleaning $(PLUGIN_NAME) plugin..."
	rm -f $(SO_FILE)
	@echo "Clean complete"

# Test compilation (without building plugin)
test:
	@echo "Testing $(PLUGIN_NAME) plugin compilation..."
	go build -o /dev/null $(MAIN_FILE)
	@echo "Compilation test passed"

# Install plugin (copy to parent plugins directory if needed)
install: build
	@echo "Plugin ready for loading: $(SO_FILE)"

.PHONY: all build clean test install 