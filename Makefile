VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
DIST := dist

.PHONY: all build test lint clean

all: build

build:
	@mkdir -p $(DIST)
	@go work sync
	@cd rook-cli && go build $(LDFLAGS) -o ../$(DIST)/rook-cli .
	@cd rook-server && go build $(LDFLAGS) -o ../$(DIST)/rook-server-cli ./cmd/admin
	@echo "Built: $(DIST)/rook-cli and $(DIST)/rook-server-cli (version: $(VERSION))"

test:
	@cd rook-cli && go test ./... -race -count=1
	@cd rook-server && go test ./... -race -count=1

lint:
	@cd rook-cli && golangci-lint run ./...
	@cd rook-server && golangci-lint run ./...

clean:
	@rm -rf $(DIST)
	@echo "Cleaned $(DIST)/"
