BIN_NAME=pb
BUILD_DIR=./dist
MAIN_PATH=./build/main.go

.PHONY: lint build build-all clean help

help:
	@echo "Usage:"
	@echo "  make build         - Build for current platform"
	@echo "  make build-all     - Build for x86_64 and arm64 (Linux & Darwin)"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make clean         - Remove build directory"

lint:
	golangci-lint run -c ./golangci.yml ./...

build:
	go build -o $(BIN_NAME) $(MAIN_PATH)

build-all: clean
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux x86_64..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)_linux_amd64 $(MAIN_PATH)
	@echo "Building for Linux arm64..."
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BIN_NAME)_linux_arm64 $(MAIN_PATH)
	@echo "Building for Darwin x86_64..."
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)_darwin_amd64 $(MAIN_PATH)
	@echo "Building for Darwin arm64..."
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BIN_NAME)_darwin_arm64 $(MAIN_PATH)
	@echo "Building for Windows x86_64..."
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BIN_NAME)_windows_amd64.exe $(MAIN_PATH)
	@echo "Done. Binaries are in $(BUILD_DIR)"

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BIN_NAME)

