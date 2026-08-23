BIN     := bin/dotfiles
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test unit release clean

build:
	go build -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/dotfiles

unit:
	go vet ./... && go test ./...

release:
	@mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build -ldflags '$(LDFLAGS)' -o dist/dotfiles-darwin-arm64 ./cmd/dotfiles
	GOOS=linux  GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o dist/dotfiles-linux-amd64 ./cmd/dotfiles
	GOOS=linux  GOARCH=arm64 go build -ldflags '$(LDFLAGS)' -o dist/dotfiles-linux-arm64 ./cmd/dotfiles

clean:
	rm -rf bin dist
