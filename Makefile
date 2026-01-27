.PHONY: all build build-backend build-frontend install clean test

# Default target
all: build

# Build both backend and frontend
build: build-backend build-frontend
	@echo "✓ Plugin built successfully"

# Build Go backend
build-backend:
	@echo "Building backend..."
	@mage -v

# Build React frontend
build-frontend:
	@echo "Building frontend..."
	@npm install
	@npm run build

# Install plugin to Grafana plugins directory
install: build
	@echo "Installing plugin..."
	@mkdir -p /var/lib/grafana/plugins/sabio-sm3-chat-plugin
	@cp -r dist/* /var/lib/grafana/plugins/sabio-sm3-chat-plugin/
	@echo "✓ Plugin installed to /var/lib/grafana/plugins/sabio-sm3-chat-plugin"
	@echo "  Restart Grafana to load the plugin:"
	@echo "  sudo systemctl restart grafana-server"

# Development mode with watch
dev:
	@npm run watch

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist/
	@rm -rf node_modules/
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running Go tests..."
	@go test ./pkg/... -v
	@echo "Running frontend tests..."
	@npm test

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod tidy
	@echo "Installing npm dependencies..."
	@npm install
	@echo "✓ Dependencies installed"

# Check if plugin is valid
validate:
	@echo "Validating plugin..."
	@npx @grafana/toolkit plugin:build --skipTest
	@echo "✓ Plugin validation complete"

# Sign plugin (required for Grafana Cloud)
sign:
	@echo "Signing plugin..."
	@npx @grafana/toolkit plugin:sign
	@echo "✓ Plugin signed"

# Package plugin for distribution
package: build
	@echo "Packaging plugin..."
	@mkdir -p releases
	@tar -czf releases/sabio-sm3-chat-plugin-$(shell cat package.json | grep version | head -1 | awk -F: '{ print $$2 }' | sed 's/[", ]//g').tar.gz -C dist .
	@echo "✓ Plugin packaged in releases/"

# Help
help:
	@echo "Available targets:"
	@echo "  make build          - Build backend and frontend"
	@echo "  make build-backend  - Build only Go backend"
	@echo "  make build-frontend - Build only React frontend"
	@echo "  make install        - Install plugin to Grafana"
	@echo "  make dev            - Start frontend in watch mode"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make test           - Run tests"
	@echo "  make deps           - Install dependencies"
	@echo "  make validate       - Validate plugin"
	@echo "  make sign           - Sign plugin for Grafana Cloud"
	@echo "  make package        - Create distribution package"
