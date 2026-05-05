BINARY    := hx
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -s -w -X main.version=$(VERSION)
GOPATH    := $(shell go env GOPATH)

.PHONY: build install clean test

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

install: build
	@mkdir -p $(GOPATH)/bin
	rm -f $(GOPATH)/bin/$(BINARY)
	cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY)
	@echo "Installed $(BINARY) to $(GOPATH)/bin/$(BINARY)"
	@echo "Add this to your .zshrc:"
	@echo '  eval "$$(hx init zsh)"'

clean:
	rm -rf bin/

test:
	go test ./...

dev: build
	./bin/$(BINARY)
