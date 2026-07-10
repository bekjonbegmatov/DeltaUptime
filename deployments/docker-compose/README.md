# docker-compose

Локальный и prod-lite стек. **Docker — единственный способ деплоя проекта**
(Kubernetes не используется).

**Базовый стек (старт):** api, scheduler, incident-worker, notification-worker,
frontend, **postgres, redis, nats**.

**Поздняя стадия** (отдельные compose-профили, вводятся при росте):
`clickhouse` (метрики при росте), `prometheus` + `grafana` (наблюдаемость).
Не включать в базовый `up` — держать за профилем, например:

```bash
docker compose up -d                      # базовый стек
docker compose --profile observability up -d   # + prometheus/grafana (поздняя стадия)
docker compose --profile clickhouse up -d       # + clickhouse (поздняя стадия)
docker compose config                     # проверить валидность перед коммитом
```

Точка старта для 50–100 агентов.
