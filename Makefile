BIN     := bin/dotfiles
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test unit release clean linux try install-bin

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
	docker run --rm -e E2E_NET=$(E2E_NET) -e E2E_BREW=$(E2E_BREW) dotfiles-e2e

# Interactive fresh-machine sandbox: a throwaway container with nothing
# installed, to walk through `dotfiles setup` by hand.
try: linux
	docker build -q -f test/docker/Dockerfile --build-arg TARGETARCH=$(DOCKER_ARCH) -t dotfiles-e2e . >/dev/null
	docker run --rm -it --entrypoint bash dotfiles-e2e test/docker/try.sh

# Install the binary for this machine into ~/.local/bin.
install-bin: build
	@mkdir -p $(HOME)/.local/bin
	cp bin/dotfiles $(HOME)/.local/bin/dotfiles
	@echo "installed $(HOME)/.local/bin/dotfiles ($$($(HOME)/.local/bin/dotfiles version))"

clean:
	rm -rf bin dist
