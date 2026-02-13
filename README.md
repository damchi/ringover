# ringover

## Requirements

- Docker + Docker Compose
- `migrate` CLI (https://github.com/golang-migrate/migrate)

## Configuration

Create a `.env` file (you can copy `.env-default`).

Example:

```env
APP_NAME=Ringover_api
APP_VERSION=DEV
APP_PORT=8080
API_HOST_PORT=8080

MYSQL_HOST=127.0.0.1
MYSQL_PORT=3306
MYSQL_HOST_PORT=3306
MYSQL_DATABASE=ringover
MYSQL_USER=ringover
MYSQL_PASSWORD=ringover
MYSQL_ROOT_PASSWORD=root
TRUSTED_PROXIES=
```

Notes:

- `MYSQL_ROOT_PASSWORD` is required for first MySQL initialization on a fresh volume.
- In Docker Compose, API DB host is forced to `db` internally.
- Leave `TRUSTED_PROXIES` empty to ignore `X-Forwarded-*` headers; set CIDR/IP list when behind a trusted reverse proxy.
- `.env` is required by the `Makefile`.

## Run

Start everything:

```bash
make start
```

This command:

1. starts Docker Compose
2. waits for MySQL health
3. runs `migrate up`
4. follows API logs

Stop log streaming with `Ctrl+C` (containers keep running).

## Useful Commands

Follow API logs:

```bash
make logs
```

Stop containers without removing them:

```bash
make stop
```

Run migrations:

```bash
make migrate-up
make migrate-down
make migrate-new name=create_example_table
```

Stop and remove everything (containers, network, volumes):

```bash
make kill
```

Run tests:

```bash
make test-unit
make test-integration
make test-all
```

## Health Endpoints

- `GET /api/health`
- `GET /api/health/report`

## Task Endpoints

- `GET /api/tasks`
- `POST /api/tasks`
- `PATCH /api/tasks/:id`
- `DELETE /api/tasks/:id`
- `GET /api/tasks/:id/subtasks`

Examples:

```bash
curl http://127.0.0.1:8080/api/tasks
curl -X POST http://127.0.0.1:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Créer endpoint POST /tasks","status":"todo","priority":2,"category_id":1}'
curl -X PATCH http://127.0.0.1:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Task updated from patch","status":"done","priority":1}'
curl -X DELETE http://127.0.0.1:8080/api/tasks/1
curl http://127.0.0.1:8080/api/tasks/1/subtasks
```

## Tests

- Unit tests: `make test-unit`
- Integration tests: `make test-integration`
- Full test run: `make test-all`

Integration tests use a dedicated MySQL test database (`*_test`) and delete it at the end of the suite.

## OpenAPI

OpenAPI specification file:

- `docs/openapi.yaml`

Quick way to visualize it:

```bash
docker run --rm -p 8082:8080 -e SWAGGER_JSON=/foo/openapi.yaml -v "$PWD/docs:/foo" swaggerapi/swagger-ui
```

Then open:

- `http://127.0.0.1:8082`

If port `8082` is already used on your machine, change the left port in `-p <host_port>:8080` (for example `-p 8090:8080`).
