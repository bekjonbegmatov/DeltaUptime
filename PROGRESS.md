# PROGRESS — журнал прогресса DeltaUptime

Хронологический журнал сделанной работы. Новые записи — сверху. Каждая запись
привязана к фазе из [ROADMAP.md](ROADMAP.md) и, по возможности, к коммиту/PR.

Формат записи:

```text
## YYYY-MM-DD — <короткий заголовок>
- Фаза: <номер и название>
- Что сделано: …
- Тесты: <какие прогнаны и результат>
- Коммит/PR: <hash или ссылка>
- Дальше: …
```

---

## 2026-07-11 — Базовый auth: Argon2id + access JWT + rotating refresh

- **Фаза:** 1 — Идентичность и мультитенантность
- **Что сделано:**
  - Добавлена миграция `00002_auth_refresh_tokens.sql` для хранения и ротации
    refresh-сессий.
  - В `sqlc` добавлены typed-запросы для `auth_refresh_tokens`; регенерирован
    пакет `packages/database/postgres`.
  - Реализован `internal/auth`: Argon2id hashing/verify, HMAC-hashed refresh
    tokens, signed access JWT, rotation refresh token при `refresh`.
  - Добавлены HTTP-роуты `POST /v1/auth/register`, `POST /v1/auth/login`,
    `POST /v1/auth/refresh`, `GET /v1/auth/me`.
  - Регистрация создаёт `user + organization + owner membership` в одной
    транзакции; `api` подключает auth-модуль при наличии `POSTGRES_DSN`.
  - `config` расширен TTL/secret-параметрами auth; `apps/control-plane/README.md`
    обновлён под актуальный статус Control Plane.
- **Тесты (все зелёные):**
  - `./.bin/sqlc compile`
  - `go build ./...`
  - `go test ./...`
  - `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --timeout=5m ./...` → `0 issues`
- **Коммит/PR:** ветка `feat/auth-jwt-refresh`.
- **Дальше:** TOTP / WebAuthn, затем разворачивать users / organizations /
  memberships и permission-based RBAC поверх базового auth.

## 2026-07-11 — sqlc-слой для users / organizations / memberships

- **Фаза:** 0→1 — фундамент данных для мультитенантности
- **Что сделано:**
  - Добавлен `sqlc.yaml`, генерация настроена напрямую от `migrations/` и
    `packages/database/queries/`.
  - Описаны первые typed-запросы для `organizations`, `users`, `memberships`:
    create/get/list под будущие auth и membership-flow.
  - Сгенерирован пакет `packages/database/postgres` под `pgx/v5`.
  - Добавлен `internal/database.Store` поверх `pgxpool` + `sqlc`-queries, чтобы
    backend-модули подключали БД через один общий entrypoint.
  - В `Makefile` добавлены `sqlc` и `sqlc-verify`; в `.gitignore` — локальные
    `.bin/` и `.cache/` для toolchain внутри репозитория.
- **Тесты (все зелёные):**
  - `./.bin/sqlc compile`
  - `go build ./...`
  - `go test ./...`
  - `golangci-lint run` через `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --timeout=5m ./...` → `0 issues`
- **Коммит/PR:** ветка `feat/db-sqlc-multitenancy`.
- **Дальше:** auth-модуль: Argon2id password hashing, регистрация/логин, JWT access +
  rotating refresh поверх нового query-layer.

## 2026-07-11 — Миграции (Goose) + первая схема PostgreSQL

- **Фаза:** 0→1 — Фундамент / мультитенантность
- **Что сделано:**
  - Зависимости: `pressly/goose/v3`, `jackc/pgx/v5` (+ go.sum).
  - Пакет `deltauptime/migrations` — встраивает `*.sql` через `//go:embed`
    (миграции едут внутри бинаря; отдельная папка при деплое не нужна).
  - `internal/database/migrate.go` — раннер goose поверх pgx (`sql.Open("pgx")`),
    ping, `UpContext`, slog-адаптер логгера. Идемпотентно.
  - Подкоманда `migrate` реально применяет миграции (заглушка убрана).
  - Первая миграция `00001_init.sql`: organizations, users, memberships
    (роль-пресеты через CHECK, каскады, индексы, extension pgcrypto).
  - Тесты: наличие встроенных миграций + требование Up/Down у каждой; `migrate`
    без DSN → понятная ошибка.
- **Тесты (все зелёные):**
  - `go build/vet/test`, golangci-lint v2.12.2 → 0 issues.
  - **E2E на реальном Postgres (docker):** `migrate` создал 4 таблицы
    (+goose_db_version), version=1; повторный запуск — no-op (идемпотентность);
    пустой DSN → exit 1.
- **Коммит/PR:** ветка `feat/db-migrations`.
- **Дальше:** sqlc-конфиг + первые запросы (orgs/users), затем auth (Argon2id).

## 2026-07-11 — CI (GitHub Actions) + подключён GitHub remote

- **Фаза:** 0 — Фундамент
- **Что сделано:**
  - Подключён remote `git@github.com:bekjonbegmatov/DeltaUptime.git`, запушен `main`.
  - `.github/workflows/ci.yml` — три job: `go` (go mod tidy check, build, vet,
    `test -race`), `lint` (golangci-lint v2.12.2), `compose` (валидация базового
    стека и всех профилей).
  - CI-бейдж в README.
  - Исправлены 3 errcheck-замечания (`fmt.Fprint*` → `_, _ =`) в `internal/app`.
- **Тесты:** локально прогнан golangci-lint v2.12.2 → `0 issues`; `go test -race`
  и `go build` зелёные; `go mod tidy` без изменений.
- **Коммит/PR:** ветка `chore/ci-pipeline` (после `feat/bootstrap...` смёржен в main).
- **Дальше:** миграции (Goose) + sqlc + первая схема PostgreSQL.

## 2026-07-11 — Каркас uptime-server + базовый docker-compose

- **Фаза:** 0 — Фундамент
- **Что сделано:**
  - Go-модуль `deltauptime` (Go 1.25, только stdlib пока).
  - Бинарь `uptime-server` с dispatch подкоманд `api|scheduler|worker|migrate|version|help`
    (`apps/control-plane/cmd/uptime-server` + `internal/app`).
  - `internal/config` — загрузка конфигурации из env с дефолтами.
  - `internal/httpapi` — HTTP-сервер: `/healthz`, `/readyz`, graceful shutdown.
  - `scheduler`/`worker` — заглушки (блокируются до сигнала), `migrate` — заглушка.
  - Unit-тесты на все три пакета (dispatch, config, health-эндпоинты).
  - `deployments/docker-compose/docker-compose.yml` — базовый стек postgres+redis+nats;
    clickhouse и prometheus/grafana за профилями `clickhouse` / `observability`.
  - `Makefile` (build/test/vet/lint/check/up/down), `.env.example` дополнен.
- **Тесты:**
  - `go build ./...`, `go vet ./...`, `go test ./...` — всё зелёное.
  - Smoke: `uptime-server version/help/unknown` (exit=1 на неизвестной команде).
  - E2E: `api` поднят, `/healthz`→`{"status":"ok"}`, `/readyz`→200, `/nope`→404,
    graceful shutdown по сигналу.
  - `docker compose config` валиден (база = 3 сервиса, с профилями = 6).
  - Поднятие стека в Docker не проверено: Docker daemon (Desktop) не запущен.
- **Коммит/PR:** ветка `feat/bootstrap-server-and-compose`.
- **Дальше:** миграции (Goose) + sqlc, схема PostgreSQL, CI (GitHub Actions).

## 2026-07-11 — Решение по инфраструктуре: только Docker, метрики/observability поздней стадией

- **Фаза:** 0 — Фундамент
- **Что сделано:** зафиксирована стадийность инфраструктуры во всех доках.
  - **Только Docker / Docker Compose. Kubernetes убран** — папка
    `deployments/kubernetes` удалена, упоминания заменены на Docker-масштабирование.
  - **Базовый стек:** PostgreSQL + Redis + NATS. Результаты проверок на старте — в
    **PostgreSQL**.
  - **ClickHouse, Prometheus, Grafana — поздняя стадия** (ClickHouse → фаза 8,
    Prometheus/Grafana → фаза 7). Держим за отдельными compose-профилями.
  - Обновлены: README, AGENTS, ROADMAP, docs/architecture, docs/database,
    deployments/docker-compose, .claude/agents/infra, .env.example, README пакетов.
- **Тесты:** документация, кода нет.
- **Коммит/PR:** `docs: simplify stack to Docker-only, defer ClickHouse/Prometheus/Grafana`.
- **Дальше:** каркас `uptime-server` + базовый docker-compose (postgres/redis/nats).

## 2026-07-11 — Инициализация проекта

- **Фаза:** 0 — Фундамент
- **Что сделано:**
  - Создана структура монорепозитория (`apps/`, `packages/`, `deployments/`,
    `migrations/`, `docs/`, `scripts/`, `.claude/agents/`).
  - Написана базовая документация: `README.md`, `AGENTS.md`, `ROADMAP.md`,
    `PROGRESS.md`, и доки в `docs/` (architecture, database, agents-protocol,
    monitors, incidents, notifications, status-pages, scheduler, security).
  - Добавлены README в каждый app/package/deployment каталог.
  - Инструкции для AI-агентов разработки в `.claude/agents/`.
  - `.gitignore`, `.env.example`.
- **Тесты:** нет кода — тестировать нечего; проверена только связность документации.
- **Коммит/PR:** `chore: scaffold project structure and docs` (initial).
- **Дальше:** Фаза 0 — Docker Compose стек и каркас `uptime-server`.
