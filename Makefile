.PHONY: build install run clean dev test vet db-up db-down db-reset db-setup setup

BINARY    = sussout
DOCKER_DB = postgresql://sussout_user:password123@localhost:5432/sussout?sslmode=disable
HAS_DOCKER := $(shell command -v docker 2>/dev/null)

build:
	go build -o $(BINARY) .

install:
	go install .

run: build
	./$(BINARY) $(ARGS)

dev:
	go run . $(ARGS)

clean:
	rm -f $(BINARY)

test:
	go test ./...

vet:
	go vet ./...

db-up:
ifndef HAS_DOCKER
	@echo "Docker not found. Skip this step if you already have PostgreSQL running."
	@echo "Set your database URL with: sussout config db <url>"
	@exit 1
endif
	docker compose up -d --wait

db-down:
ifndef HAS_DOCKER
	@echo "Docker not found."
	@exit 1
endif
	docker compose down

db-reset:
ifndef HAS_DOCKER
	@echo "Docker not found."
	@exit 1
endif
	docker compose down -v
	docker compose up -d --wait

db-migrate:
	@test -n "$$DATABASE_URL" || (echo "DATABASE_URL is not set" && exit 1)
	@psql "$$DATABASE_URL" -f db/schema.sql

setup: install
ifdef HAS_DOCKER
	@echo ""
	@echo "Starting Docker PostgreSQL..."
	@docker compose up -d --wait 2>/dev/null && echo "  Database ready at $(DOCKER_DB)" || true
endif
	@echo ""
	@echo "Setup complete."
	@echo ""
	@echo "Configure your database URL:"
	@echo "  sussout config db <postgresql://user:pass@host:5432/dbname>"
ifdef HAS_DOCKER
	@echo "  Or for the local Docker instance:"
	@echo "  sussout config db \"$(DOCKER_DB)\""
endif
	@echo ""
	@echo "Then add an LLM server and start:"
	@echo "  sussout config add my-server"
	@echo "  sussout start"