.PHONY: build install run clean dev test vet db-up db-down db-reset db-setup setup

BINARY    = sussout
DOCKER_DB = postgresql://sussout_user:password123@localhost:5432/sussout?sslmode=disable

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
	docker compose up -d --wait

db-down:
	docker compose down

db-reset:
	docker compose down -v
	docker compose up -d --wait

db-migrate:
	@test -n "$$DATABASE_URL" || (echo "DATABASE_URL is not set" && exit 1)
	@psql "$$DATABASE_URL" -f db/schema.sql

db-setup: db-up
	@echo "PostgreSQL is ready."
	@echo ""
	@echo "Add this to your .env file:"
	@echo "  DATABASE_URL=$(DOCKER_DB)"

setup: db-setup install
	@echo ""
	@echo "Setup complete. Run 'sussout config add' then 'sussout start'."