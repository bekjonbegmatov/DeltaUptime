# docker-compose

Локальный и prod-lite стек. Сервисы: api, scheduler, incident-worker,
notification-worker, frontend, postgres, clickhouse, redis, nats, prometheus,
grafana.

```bash
docker compose up -d      # поднять инфраструктуру
docker compose config     # проверить валидность перед коммитом
```

Точка старта для 50–100 агентов (Kubernetes пока не нужен).
