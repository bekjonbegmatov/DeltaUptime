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
