BIN := bin/si
INSTALL_DIR ?= $(HOME)/.bin

.PHONY: docs-site build test install play-example examples check fmt clean studio-ui studio-e2e desktop-mac desktop-icon desktop-cert

build:
	go build -o $(BIN) ./cmd/si

test:
	gofmt -l . | (! grep .)
	go vet ./...
	go test ./...

install: build
	@mkdir -p "$(INSTALL_DIR)"
	@rm -f "$(INSTALL_DIR)/si" && cp "$(BIN)" "$(INSTALL_DIR)/si" # new inode: macOS kills a binary overwritten in place
	@echo "installed $(INSTALL_DIR)/si"
	@case ":$$PATH:" in \
	  *":$(INSTALL_DIR):"*) echo "$(INSTALL_DIR) is already on your PATH";; \
	  *) rc="$$HOME/.bashrc"; case "$$SHELL" in *zsh) rc="$$HOME/.zshrc";; esac; \
	     printf 'Add %s to PATH in %s? [y/N] ' "$(INSTALL_DIR)" "$$rc"; read ans; \
	     if [ "$$ans" = y ] || [ "$$ans" = Y ]; then \
	       printf '\nexport PATH="%s:$$PATH"\n' "$(INSTALL_DIR)" >> "$$rc"; \
	       echo "added to $$rc — run: source $$rc"; \
	     else \
	       echo 'not added; add this to your shell rc:'; \
	       echo '  export PATH="$(INSTALL_DIR):$$PATH"'; \
	     fi;; \
	esac

play-example: build
	$(BIN) play examples --open

examples: build
	$(BIN) --output out/examples examples

check: build
	$(BIN) check examples

fmt:
	gofmt -w .

# rebuild the embedded Studio UI (needs Node; the built bundle is committed)
studio-ui:
	cd web/studio && npm ci && npm test && npm run build

clean:
	rm -rf bin out html

studio-e2e: build
	cd web/studio && npm run e2e

# si Studio.app for macOS (universal), into desktop/dist; needs Xcode command line tools
desktop-mac:
	desktop/build-mac.sh

# desktop/AppIcon.icns from desktop/icon/gen.go (the Stack mark); macOS only (sips, iconutil)
desktop-icon:
	cd desktop && rm -rf /tmp/si-icon.iconset && mkdir /tmp/si-icon.iconset && go run icon/gen.go /tmp/si-icon.png && \
	for s in 16 32 128 256 512; do sips -z $$s $$s /tmp/si-icon.png --out /tmp/si-icon.iconset/icon_$${s}x$${s}.png >/dev/null && \
	sips -z $$((s*2)) $$((s*2)) /tmp/si-icon.png --out /tmp/si-icon.iconset/icon_$${s}x$${s}@2x.png >/dev/null; done && \
	iconutil -c icns /tmp/si-icon.iconset -o AppIcon.icns

# the docs site, built with si from RULES.md (generated pages) into dist/docs-site
docs-site:
	rm -rf dist/docs-site-src dist/docs-site
	go run ./cmd/si --export docs-site --output dist/docs-site-src
	go run ./cmd/si build dist/docs-site-src --output dist/docs-site --share-assets

# a self-signed code-signing identity in your login keychain, once, for desktop-mac
desktop-cert:
	desktop/make-cert.sh
