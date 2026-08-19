.PHONY: build install run clean dev test vet setup _ensure-go

BINARY = sussout

_ensure-go:
	@command -v go >/dev/null 2>&1 || { \
		echo "Go is not installed."; \
		if command -v brew >/dev/null 2>&1; then \
			echo "Installing Go via Homebrew..."; \
			brew install go; \
		else \
			echo "Please install Go: https://go.dev/dl/"; \
			exit 1; \
		fi; \
	}

build: _ensure-go
	go build -o $(BINARY) .

install: _ensure-go
	go install .

run: build
	./$(BINARY) $(ARGS)

dev: _ensure-go
	go run . $(ARGS)

clean: _ensure-go
	rm -f $(BINARY)

test: _ensure-go
	go test ./...

vet: _ensure-go
	go vet ./...

setup: install
	@echo ""
	@echo "Setup complete. SQLite is the default database (no config needed)."
	@echo ""
	@GOPATH=$$(go env GOPATH); \
	BINDIR="$$GOPATH/bin"; \
	if ! echo "$$PATH" | tr ':' '\n' | grep -qxF "$$BINDIR"; then \
		echo "IMPORTANT: Go binaries are not in your PATH."; \
		echo ""; \
		SHELL_NAME=$$(basename "$$SHELL"); \
		case "$$SHELL_NAME" in \
			zsh)  RCFILE="$$HOME/.zshrc" ;; \
			bash) RCFILE="$$HOME/.bash_profile" ;; \
			*)    RCFILE="your shell profile" ;; \
		esac; \
		echo "Add this line to $$RCFILE:"; \
		echo "  export PATH=\"$$BINDIR:\$$PATH\""; \
		echo ""; \
		echo "Then run: source $$RCFILE"; \
		echo ""; \
	fi
	@echo "Add an LLM server and start:"
	@echo "  sussout config add my-server"
	@echo "  sussout start"