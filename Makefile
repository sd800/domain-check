PREFIX ?= $(HOME)/.local
GO ?= go

.PHONY: build test check security install uninstall dist clean

build:
	$(GO) build -trimpath -o build/bin/domain ./cmd/domain

test:
	$(GO) test -race ./...

check:
	$(GO) vet ./...
	$(GO) test -race ./...
	@test -z "$$(gofmt -l cmd internal version.go)" || (echo 'Run gofmt before committing.'; exit 1)

security:
	$(GO) mod verify
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
	python3 scripts/test_install.py

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 755 build/bin/domain "$(DESTDIR)$(PREFIX)/bin/domain"

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/domain"

dist:
	sh scripts/dist.sh

clean:
	rm -rf build coverage.out
