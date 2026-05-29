PREFIX ?= /usr/local
BINDIR  = $(PREFIX)/bin

.PHONY: build install uninstall clean

build:
	go build -o bin/userproxyportal ./cmd/userproxyportal

install: build
	install -Dm755 bin/userproxyportal $(BINDIR)/userproxyportal
	@echo "→ Installez le programme: userproxyportal install"

uninstall:
	rm -f $(BINDIR)/userproxyportal

clean:
	rm -rf bin/
