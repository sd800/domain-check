PREFIX ?= $(HOME)/.local
GO ?= go

.PHONY: build test check security install uninstall dist clean

build:
	$(GO) build -trimpath -o build/bin/domain-check ./cmd/domain
	@if [ "$$(uname -s)" = Darwin ]; then /usr/bin/codesign --force --sign - --identifier localhost.domain-check build/bin/domain-check; fi
	ln -sfn domain-check build/bin/domain

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
	sh scripts/install.sh --binary-only build/bin/domain-check "$(DESTDIR)$(PREFIX)/bin"
	@if [ -z "$(DESTDIR)" ]; then "$(PREFIX)/bin/domain" --web-refresh; fi

uninstall:
	"$(DESTDIR)$(PREFIX)/bin/domain-check" uninstall

dist:
	sh scripts/dist.sh

clean:
	rm -rf build coverage.out
