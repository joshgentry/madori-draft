.PHONY: build build-release build-verify clean test vet tidy syso

APP_NAME = madori
VERSION = 1.0.0
BUILD_DIR = build
WINRES_JSON = resources/winres.json
SYSO_OUT = cmd/madori
GO_WINRES = go tool go-winres
ICON_FILE = resources/pwIcon.ico

# Default: build Windows .exe with console (useful for debugging)
build: syso
	GOOS=windows GOARCH=amd64 go build \
		-ldflags="-X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME).exe \
		./cmd/madori/

# Quick compile check — builds to build/ then removes the binary.
# Succeeds only if compilation and link complete without errors.
build-verify:
	gofmt -w -s .
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-test-build.exe ./cmd/madori/ && rm $(BUILD_DIR)/$(APP_NAME)-test-build.exe && echo "Build OK"

# Release: build Windows .exe without console window (gui subsystem)
build-release: syso
	GOOS=windows GOARCH=amd64 go build \
		-ldflags="-H windowsgui -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME).exe \
		./cmd/madori/

# Always regenerate .syso files — removes stale ones first, then runs go-winres.
# Depends on winres.json and the icon file so any change to either triggers a rebuild.
syso: $(WINRES_JSON) $(ICON_FILE)
	@rm -f $(SYSO_OUT)/rsrc_windows_*.syso
	cd $(SYSO_OUT) && $(GO_WINRES) make --in ../../$(WINRES_JSON)

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
