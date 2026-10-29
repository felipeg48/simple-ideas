BIN := bin/si
INSTALL_DIR ?= $(HOME)/.bin

.PHONY: build test install play-example examples check fmt clean studio-ui studio-e2e

build:
	go build -o $(BIN) ./cmd/si

test:
	gofmt -l . | (! grep .)
	go vet ./...
	go test ./...

install: build
	@mkdir -p "$(INSTALL_DIR)"
	@cp "$(BIN)" "$(INSTALL_DIR)/si"
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
