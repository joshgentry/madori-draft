.PHONY: build build-release clean test vet tidy syso

APP_NAME = durablewindows
VERSION = 1.0.0
BUILD_DIR = build
WINRES_JSON = resources/winres.json
SYSO_OUT = cmd/durablewindows
GO_WINRES = go tool go-winres

# Default: build Windows .exe with console (useful for debugging)
build: $(SYSO_OUT)/rsrc_windows_amd64.syso
	GOOS=windows GOARCH=amd64 go build \
		-ldflags="-X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME).exe \
		./cmd/durablewindows/

# Release: build Windows .exe without console window (gui subsystem)
build-release: $(SYSO_OUT)/rsrc_windows_amd64.syso
	GOOS=windows GOARCH=amd64 go build \
		-ldflags="-H windowsgui -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME).exe \
		./cmd/durablewindows/

# Generate Windows resource .syso files (VERSIONINFO, icons, manifest)
$(SYSO_OUT)/rsrc_windows_amd64.syso: $(WINRES_JSON)
	cd $(SYSO_OUT) && $(GO_WINRES) make --in ../../$(WINRES_JSON)

# Convenience target: generate .syso files without building
syso: $(SYSO_OUT)/rsrc_windows_amd64.syso

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(SYSO_OUT)/rsrc_windows_*.syso

# Compile and run tests (execution only works on Windows)
test:
	GOOS=windows GOARCH=amd64 go test ./...

# Static analysis (unsafe.Pointer warnings in Win32 interop are expected)
vet:
	-GOOS=windows GOARCH=amd64 go vet ./...

# Tidy dependencies
tidy:
	go mod tidy
