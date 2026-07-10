# migrations

SQL-миграции PostgreSQL (Goose). Именование: `NNNNN_description.sql` с секциями
`-- +goose Up` / `-- +goose Down`.

```bash
uptime-server migrate         # применить все
```

Метрики проверок живут в ClickHouse — их DDL в [`../packages/database/`](../packages/database/),
не здесь.
