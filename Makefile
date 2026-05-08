.PHONY: build test install uninstall

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/cclmonitor ./cmd/cclmonitor
	go build -o bin/cclmonitor-install ./cmd/cclmonitor-install
	go build -o bin/cclmonitor-tail ./cmd/cclmonitor-tail

test:
	go test ./...

BINDIR := $(HOME)/bin

install: build
	mkdir -p $(BINDIR)
	cp bin/cclmonitor $(BINDIR)/cclmonitor
	cp bin/cclmonitor-install $(BINDIR)/cclmonitor-install
	cp bin/cclmonitor-tail $(BINDIR)/cclmonitor-tail
	$(BINDIR)/cclmonitor-install
	@echo ""
	@echo "------------------------------------------------------"
	@echo "次の行を ~/.zshrc に追加すると cclmonitor-tail などを"
	@echo "ターミナルから直接実行できます："
	@echo "  export PATH=\"\$$HOME/bin:\$$PATH\""
	@echo "追加後: source ~/.zshrc"
	@echo "------------------------------------------------------"

uninstall:
	rm -f $(BINDIR)/cclmonitor
	rm -f $(BINDIR)/cclmonitor-install
	rm -f $(BINDIR)/cclmonitor-tail
	@if [ -f ~/.claude/settings.json.bak ]; then \
		cp ~/.claude/settings.json.bak ~/.claude/settings.json; \
		echo "settings.json restored from backup"; \
	else \
		echo "no backup found, settings.json unchanged"; \
	fi
