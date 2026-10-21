PREFIX ?= $(HOME)/.local
GO ?= go

.PHONY: build test check install uninstall dist clean

build:
	$(GO) build -trimpath -o bin/domain ./cmd/domain

test:
	$(GO) test -race ./...

check:
	$(GO) vet ./...
	$(GO) test -race ./...
	@test -z "$$(gofmt -l cmd internal version.go)" || (echo 'Run gofmt before committing.'; exit 1)

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 755 bin/domain "$(DESTDIR)$(PREFIX)/bin/domain"

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/domain"

dist:
	sh scripts/dist.sh

clean:
	rm -rf bin dist coverage.out
