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
