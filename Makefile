MIGRATE ?= migrate

ENV_FILE := .env

ifeq (,$(wildcard $(ENV_FILE)))
$(error $(ENV_FILE) file not found)
endif

include $(ENV_FILE)
export $(shell sed 's/=.*//' $(ENV_FILE))

REQUIRED_VARS := MYSQL_USER MYSQL_PASSWORD MYSQL_ROOT_PASSWORD MYSQL_HOST MYSQL_PORT MYSQL_DATABASE
$(foreach v,$(REQUIRED_VARS),$(if $($(v)),,$(error Missing $(v) in $(ENV_FILE))))

MYSQL_DSN := mysql://$(MYSQL_USER):$(MYSQL_PASSWORD)@tcp($(MYSQL_HOST):$(MYSQL_PORT))/$(MYSQL_DATABASE)
ifneq ($(strip $(MYSQL_PARAMS)),)
MYSQL_DSN := $(MYSQL_DSN)?$(MYSQL_PARAMS)
endif

.PHONY: check-requirements start logs stop kill migrate-new migrate-up migrate-down

check-requirements:
	@command -v docker >/dev/null 2>&1 || { echo "ERROR: docker is not installed."; exit 1; }
	@docker compose version >/dev/null 2>&1 || { echo "ERROR: docker compose is not available."; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "ERROR: docker daemon is not running."; exit 1; }
	@command -v "$(MIGRATE)" >/dev/null 2>&1 || { echo "ERROR: $(MIGRATE) is not installed (required for migrations)."; exit 1; }

# Start DB first, wait until healthy, apply migrations, then start API.
start: check-requirements
	@docker compose up -d db
	@for i in $$(seq 1 60); do \
		status=$$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' ringover-db 2>/dev/null || true); \
		if [ "$$status" = "healthy" ]; then \
			break; \
		fi; \
		sleep 1; \
	done; \
	if [ "$$status" != "healthy" ]; then \
		echo "ERROR: database did not become healthy in time"; \
		exit 1; \
	fi
	@$(MAKE) migrate-up
	@docker compose up -d api
	@echo "Everything started successfully: docker compose is up and migrations are applied."
	@echo "Following API logs (Ctrl+C to stop logs, containers stay running)..."
	@$(MAKE) --no-print-directory logs

logs:
	@docker compose logs -f api

stop:
	@docker compose stop
	@echo "Containers are stopped (not removed)."

kill:
	@docker compose down -v --remove-orphans
	@echo "Everything is stopped and removed (containers, network, volumes)."

migrate-new:
	@if [ -z "$(name)" ]; then \
		echo "ERROR: Please provide a name, e.g. make migrate-new name=create_users_table"; \
		exit 1; \
	fi
	$(MIGRATE) create -ext sql -dir db/migrations $(name)

migrate-up:
	$(MIGRATE) -database '$(MYSQL_DSN)' -path db/migrations up

migrate-down:
	$(MIGRATE) -database '$(MYSQL_DSN)' -path db/migrations down
