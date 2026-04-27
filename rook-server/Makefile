VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
DIST := ../dist

.PHONY: build test lint clean

build:
	@mkdir -p $(DIST)
	@go build $(LDFLAGS) -o $(DIST)/rook-server-cli ./cmd/admin
	@echo "Built: $(DIST)/rook-server-cli"

test:
	@go test ./... -race -count=1

lint:
	@golangci-lint run ./...

clean:
	@rm -f $(DIST)/rook-server-cli
