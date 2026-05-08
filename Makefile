APP_NAME    := iodat
VERSION     := 1.0.0
BUILD_DIR   := build

.PHONY: all windows linux darwin clean

all: windows linux darwin

windows:
	@echo "Compilando $(APP_NAME) v$(VERSION) para Windows (amd64)..."
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-windows-amd64.exe ./cmd/$(APP_NAME)/
	@echo "  → $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-windows-amd64.exe"

linux:
	@echo "Compilando $(APP_NAME) v$(VERSION) para Linux (amd64)..."
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-linux-amd64 ./cmd/$(APP_NAME)/
	@echo "  → $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-linux-amd64"

darwin:
	@echo "Compilando $(APP_NAME) v$(VERSION) para macOS (amd64)..."
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-darwin-amd64 ./cmd/$(APP_NAME)/
	@echo "  → $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-darwin-amd64"
	@echo "Compilando $(APP_NAME) v$(VERSION) para macOS (arm64)..."
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-darwin-arm64 ./cmd/$(APP_NAME)/
	@echo "  → $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-darwin-arm64"

build: linux
	@echo "Build local completado."

clean:
	rm -rf $(BUILD_DIR)/
	@echo "Limpiado."

test:
	go test ./...

.PHONY: verify
verify:
	@echo "=== SHA256SUMS v$(VERSION) ===" > $(BUILD_DIR)/SHA256SUMS
	@for bin in $(BUILD_DIR)/*; do \
		if [ -f "$$bin" ] && [ "$$bin" != "$(BUILD_DIR)/SHA256SUMS" ]; then \
			shasum -a 256 "$$bin" | tee -a $(BUILD_DIR)/SHA256SUMS; \
		fi; \
	done
	@echo ""
	@echo "SHA256SUMS generado en $(BUILD_DIR)/SHA256SUMS"

.PHONY: sbom
sbom:
	@echo "Software Bill of Materials — $(APP_NAME) v$(VERSION)"
	@echo "========================================="
	@go version -m $(BUILD_DIR)/$(APP_NAME)-v$(VERSION)-linux-amd64 2>/dev/null || go version -m ./$(APP_NAME) 2>/dev/null || echo "Compila primero con 'make build' o 'make all'"
	@echo ""
	@echo "Dependencias externas:"
	@go list -m all 2>/dev/null || true
	@echo ""
	@echo "Archivos fuente:"
	@find . -name '*.go' -not -path './build/*' -not -path './.git/*' | sort

.PHONY: run
run:
	go run ./cmd/$(APP_NAME)/
