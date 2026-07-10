# Архитектура DeltaUptime

## Общий подход

**Модульный монолит Control Plane + отдельные агенты + отдельные worker-процессы.**
Не 15 микросервисов сразу и не Kubernetes-зоопарк. Один backend-проект с чёткими
внутренними модулями; при росте нагрузки `scheduler`, `notifications` и
`incident engine` выносятся в отдельные сервисы.

```text
control-plane/
├── auth            ├── scheduler       ├── notifications
├── users           ├── checks          ├── status-pages
├── organizations   ├── incidents       ├── billing
├── agents          ├── monitors        ├── audit
└── integrations
```

## Поток данных

```text
Web Panel ──HTTPS──▶ Control Plane ──tasks──▶ NATS ──▶ Probe agents
                          ▲                              │
                          └────── results / events ──────┘
                                                         │
Probe agents ──────────── метрики ──────────────▶ ClickHouse
Control Plane ─── конфигурация / сущности ──────▶ PostgreSQL
```

## Один бинарь — разные процессы

```bash
uptime-server migrate     # миграции (Goose)
uptime-server api         # HTTP API + realtime gateway (SSE)
uptime-server scheduler   # планировщик (advisory locks / leader election)
uptime-server worker      # incident engine + notification workers
```

Один код, разные команды — на старте проще, чем множество сервисов.

## Стек backend

| Назначение | Выбор |
|-----------|-------|
| Язык | Go (производительность, сеть, конкурентность, один бинарь) |
| HTTP API | Chi (или Echo) |
| Доступ к БД | pgx + sqlc (без тяжёлого ORM — контролируемые запросы) |
| Миграции | Goose |
| Валидация | go-playground/validator |
| Конфиг | envconfig / koanf |
| Логи | slog / zap |
| Трейсинг | OpenTelemetry |
| gRPC | grpc-go |
| WebSocket | nhooyr/websocket |

## Два раздельных frontend

```text
apps/panel-web   → panel.domain.com   (приватная админ-панель, авторизация, realtime)
apps/status-web  → status.domain.com  (публичные status pages, SSR + CDN, Custom CSS)
```

Разделение критично: если основная панель временно упадёт — публичные status pages
продолжают отдаваться из кэша. Пользовательский Custom CSS на status-web физически
не может затронуть админ-панель.

## Масштабирование

- **Старт:** 1×(CP, PostgreSQL, ClickHouse, Redis, NATS), 50–100 агентов.
- **Средний:** 2–3 API, 2 scheduler, несколько workers, PG primary+replica,
  ClickHouse cluster, NATS cluster.
- **Большой:** региональные ingress, agent gateway, отдельные scheduler/incident/
  notification сервисы, PostgreSQL HA.

Для 50 агентов Kubernetes не обязателен — начинаем с Docker Compose.

## Наблюдаемость самой платформы

Prometheus + Grafana + Loki + Tempo + OpenTelemetry. Ключевая метрика —
`scheduler_lag_seconds` (насколько проверка запустилась позже запланированного).
Полный список метрик — в [ROADMAP.md](../ROADMAP.md), фаза 7.
