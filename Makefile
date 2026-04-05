BINARY_NAME := craft
BIN_DIR     := bin
MAIN_FILE   := .

# --- OS Detection ---
ifeq ($(OS),Windows_NT)
	EXT := .exe
	RM_CMD := del /Q /S
	MKDIR_CMD := if not exist "$(BIN_DIR)" mkdir "$(BIN_DIR)"
else
	EXT :=
	RM_CMD := rm -rf
	MKDIR_CMD := mkdir -p $(BIN_DIR)
endif

LDFLAGS := -s -w

.PHONY: all build install clean format vet test release help

all: format vet build


build:
	@echo "[BUILD] Compiling Craft Engine..."
	@$(MKDIR_CMD)
	@go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)$(EXT) $(MAIN_FILE)
	@echo "[BUILD] Success! Binary is located at $(BIN_DIR)/$(BINARY_NAME)$(EXT)"


install: format vet
	@echo "[INSTALL] Installing Craft globally (to GOPATH/bin)..."
	@go install -ldflags="$(LDFLAGS)"
	@echo "[INSTALL] Craft is now installed globally! Open a new terminal and type 'craft'."


format:
	@echo "[FMT] Formatting code..."
	@go fmt ./...

vet:
	@echo "[VET] Running static analysis..."
	@go vet ./...

test:
	@echo "[TEST] Running tests..."
	@go test ./... -v

clean:
	@echo "[CLEAN] Removing build artifacts..."
	@$(RM_CMD) $(BIN_DIR)
	@echo "[CLEAN] Done."

release: clean
	@echo "[RELEASE] Cross-compiling for multiple platforms..."
	@$(MKDIR_CMD)
	@echo " -> Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)
	@echo " -> Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)
	@echo " -> Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)
	@echo " -> Building for macOS (Apple Silicon arm64)..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_FILE)
	@echo "[RELEASE] All binaries generated successfully in $(BIN_DIR)/"

help:
	@echo "==============================================="
	@echo " CRAFT ENGINE - REPOSITORY MAKEFILE"
	@echo "The Makefile structure has been included for illustrative purposes only and should not be used to build Craft; you can use Craft to build Craft itself."
	@echo "==============================================="
	@echo " Commands:"
	@echo "   make build    - Compile the project into ./bin"
	@echo "   make install  - Install globally (go install) so you can use 'craft' anywhere"
	@echo "   make format   - Format Go code"
	@echo "   make vet      - Run Go vet for static analysis"
	@echo "   make test     - Run tests"
	@echo "   make release  - Compile for Linux, Windows, and macOS (Cross-compile)"
	@echo "   make clean    - Remove build artifacts"