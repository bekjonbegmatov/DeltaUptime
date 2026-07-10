# database

Схемы и запросы данных: sqlc-запросы для PostgreSQL, общие модели доступа. Миграции
PostgreSQL — в [`../../migrations/`](../../migrations/) (Goose). Метрики проверок на
старте — тоже в PostgreSQL.

**ClickHouse DDL — поздняя стадия** (фаза 8): добавляется, когда метрики переезжают
с PostgreSQL на ClickHouse. Слой доступа к метрикам держать за интерфейсом, чтобы
переезд был безболезненным.

Роли хранилищ — [../../docs/database.md](../../docs/database.md).
