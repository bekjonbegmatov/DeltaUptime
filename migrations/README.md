# migrations

SQL-миграции PostgreSQL (Goose). Именование: `NNNNN_description.sql` с секциями
`-- +goose Up` / `-- +goose Down`.

Это одновременно Go-пакет `deltauptime/migrations`: `migrations.go` встраивает все
`*.sql` через `//go:embed`, поэтому они едут внутри бинаря `uptime-server` — при
деплое отдельная папка миграций не нужна.

```bash
uptime-server migrate         # применить все (идемпотентно)
```

Реализация раннера — [`../apps/control-plane/internal/database/migrate.go`](../apps/control-plane/internal/database/migrate.go).
Каждая миграция обязана иметь и `Up`, и `Down` (проверяется тестом).

Метрики проверок на старте — в PostgreSQL; ClickHouse DDL (поздняя стадия) будет в
[`../packages/database/`](../packages/database/), не здесь.
