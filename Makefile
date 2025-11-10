# Makefile for SimpleC2 Framework

# --- Configurable Variables ---

# Default listener URL to be embedded in the beacons.
# Can be overridden from the command line, e.g., `make beacons-http LISTENER_URL=http://1.2.3.4:8080`
LISTENER_URL ?= http://localhost:8888

# --- Build Configuration ---

# Go build flags
LDFLAGS_VAR = main.serverURL
# Add -s -w to strip binaries
LDFLAGS = -ldflags="-X '$(LDFLAGS_VAR)=$(LISTENER_URL)' -s -w"

# Component binary names
BINARY_TS = teamserver
BINARY_LISTENER_HTTP = listener_http
BEACON_HTTP_WIN = beacon_http.exe
BEACON_HTTP_LINUX = beacon_http.linux
BEACON_HTTP_DARWIN = beacon_http.darwin

# --- Directory and Path Configuration ---
BIN_DIR := bin
TS_DIR := $(BIN_DIR)/teamserver
LISTENER_HTTP_DIR := $(BIN_DIR)/listener_http
BEACONS_DIR := $(BIN_DIR)/beacons

BINARY_TS_PATH := $(TS_DIR)/$(BINARY_TS)
BINARY_LISTENER_HTTP_PATH := $(LISTENER_HTTP_DIR)/$(BINARY_LISTENER_HTTP)
BEACON_WIN_PATH := $(BEACONS_DIR)/$(BEACON_HTTP_WIN)
BEACON_LINUX_PATH := $(BEACONS_DIR)/$(BEACON_HTTP_LINUX)
BEACON_DARWIN_PATH := $(BEACONS_DIR)/$(BEACON_HTTP_DARWIN)

# --- Build Targets ---

.PHONY: all
all: teamserver http
	@echo "All components built successfully into separate directories in $(BIN_DIR)/"

.PHONY: http
http: listener-http beacons-http
	@echo "HTTP listener and beacons built successfully."

.PHONY: teamserver
teamserver: $(BINARY_TS_PATH)

$(BINARY_TS_PATH): teamserver/*.go teamserver/**/*.go
	@mkdir -p $(TS_DIR)/loot
	@mkdir -p $(TS_DIR)/uploads
	@mkdir -p $(TS_DIR)/certs
	@echo "Building TeamServer into $(TS_DIR)/"
	go build -ldflags="-s -w" -o $@ ./teamserver

.PHONY: listener-http
listener-http: $(BINARY_LISTENER_HTTP_PATH)

$(BINARY_LISTENER_HTTP_PATH): listeners/http/*.go
	@mkdir -p $(LISTENER_HTTP_DIR)
	@mkdir -p $(LISTENER_HTTP_DIR)/certs
	@echo "Building HTTP Listener into $(LISTENER_HTTP_DIR)/"
	go build -ldflags="-s -w" -o $@ ./listeners/http

.PHONY: beacons-http
beacons-http: $(BEACON_LINUX_PATH) $(BEACON_WIN_PATH) $(BEACON_DARWIN_PATH)
	@echo "All HTTP beacons built successfully into $(BEACONS_DIR)/"

$(BEACON_LINUX_PATH):
	@mkdir -p $(BEACONS_DIR)
	@echo "Building Linux Beacon (HTTP)..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $@ ./agents/http

$(BEACON_WIN_PATH):
	@mkdir -p $(BEACONS_DIR)
	@echo "Building Windows Beacon (HTTP)..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $@ ./agents/http

$(BEACON_DARWIN_PATH):
	@mkdir -p $(BEACONS_DIR)
	@echo "Building Darwin (macOS) Beacon (HTTP)..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $@ ./agents/http

# --- Utility Targets ---

.PHONY: generate-keys
generate-keys:
	@echo "Generating all cryptographic materials (E2E keys and mTLS certs)..."
	@go run ./scripts/generate-keys.Go

.PHONY: cp-certs
cp-certs:
	@echo "Copy certs..."
	@cp -f ./certs/teamserver/* ./bin/teamserver/certs/
	@cp -f ./certs/listener/* ./bin/listener_http/certs/

.PHONY: clean
clean:
	@echo "Cleaning up build artifacts..."
	@rm -rf $(BIN_DIR)

.PHONY: run-teamserver
run-teamserver:
	@echo "Running TeamServer..."
	go run ./teamserver

.PHONY: run-listener-http
run-listener-http:
	@echo "Running HTTP Listener..."
	go run ./listeners/http

.PHONY: run-beacon-http
run-beacon-http:
	@echo "Running HTTP Beacon for development..."
	go run $(LDFLAGS) ./agents/http