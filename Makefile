BIN     := bin/dotfiles
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test unit release clean linux

build:
	go build -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/dotfiles

unit:
	go vet ./... && go test ./...

release:
	@mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build -ldflags '$(LDFLAGS)' -o dist/dotfiles-darwin-arm64 ./cmd/dotfiles
	GOOS=linux  GOARCH=amd64 go build -ldflags '$(LDFLAGS)' -o dist/dotfiles-linux-amd64 ./cmd/dotfiles
	GOOS=linux  GOARCH=arm64 go build -ldflags '$(LDFLAGS)' -o dist/dotfiles-linux-arm64 ./cmd/dotfiles

linux:
	@mkdir -p dist
	GOOS=linux GOARCH=$(DOCKER_ARCH) go build -ldflags '$(LDFLAGS)' -o dist/dotfiles-linux-$(DOCKER_ARCH) ./cmd/dotfiles

# End-to-end install/uninstall cycle in a throwaway container (Colima/Docker).
DOCKER_ARCH ?= $(shell docker version --format '{{.Server.Arch}}' 2>/dev/null || echo amd64)
test: unit linux
	docker build -q -f test/docker/Dockerfile --build-arg TARGETARCH=$(DOCKER_ARCH) -t dotfiles-e2e . >/dev/null
	docker run --rm -e E2E_NET=$(E2E_NET) dotfiles-e2e

clean:
	rm -rf bin dist
