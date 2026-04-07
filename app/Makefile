.PHONY: api-dev api-build api-test migrate-up migrate-down migrate-create lint clean

# ─── Go API ─────────────────────────────────────────────────
API_DIR := services/api
GO := go
GOFLAGS := -v

api-dev:
	cd $(API_DIR) && $(GO) run ./cmd/server

api-build:
	cd $(API_DIR) && CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o ../../bin/api-server ./cmd/server

api-test:
	cd $(API_DIR) && $(GO) test ./... -race -cover

api-lint:
	cd $(API_DIR) && golangci-lint run ./...

# ─── Database ───────────────────────────────────────────────
DB_URL ?= postgres://localhost:5432/treasury?sslmode=disable

migrate-up:
	migrate -database "$(DB_URL)" -path $(API_DIR)/migrations up

migrate-down:
	migrate -database "$(DB_URL)" -path $(API_DIR)/migrations down 1

migrate-create:
	@read -p "Migration name: " name; \
	touch $(API_DIR)/migrations/$$(date +%Y%m%d%H%M%S)_$$name.up.sql; \
	touch $(API_DIR)/migrations/$$(date +%Y%m%d%H%M%S)_$$name.down.sql; \
	echo "Created migration: $$name"

# ─── Web ────────────────────────────────────────────────────
web-dev:
	pnpm --filter @treasury/web dev

web-build:
	pnpm --filter @treasury/web build

# ─── Full stack ─────────────────────────────────────────────
dev:
	@echo "Khởi động toàn bộ services..."
	$(MAKE) api-dev &
	pnpm dev

build:
	$(MAKE) api-build
	pnpm build

clean:
	rm -rf bin/
	pnpm clean

lint:
	$(MAKE) api-lint
	pnpm lint

# ─── Docker ─────────────────────────────────────────────────
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down
