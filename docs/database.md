# Данные: PostgreSQL, Redis, NATS (+ ClickHouse на поздней стадии)

Нельзя использовать одну базу для всего. У каждого хранилища — своя роль.

> **Стадийность.** Базовый стек — **PostgreSQL + Redis + NATS** в Docker.
> Результаты проверок на старте пишутся в **PostgreSQL**. **ClickHouse вводится
> позже** (фаза 8), когда объём метрик перерастёт PostgreSQL.

## PostgreSQL — источник истины

Хранит конфигурацию и основные сущности:

- users, organizations, memberships, roles
- auth refresh tokens, TOTP secrets, WebAuthn credentials/sessions
- agents, agent groups
- monitors, notification channels, escalation policies
- incidents, maintenance windows
- status pages, API keys, audit log
- subscriptions, limits

Доступ — через pgx + sqlc (типобезопасные запросы, без ORM). Миграции — Goose,
в каталоге [`../migrations/`](../migrations/).

## Метрики проверок: PostgreSQL сейчас → ClickHouse потом

**На старте** результаты проверок (latency, status code, DNS/TLS/TCP timings,
response size, error category, agent region, monitor id, timestamp) хранятся в
**PostgreSQL** — с продуманной схемой, индексами по (monitor_id, timestamp),
партиционированием по времени и фоновой агрегацией/ретеншеном. Этого достаточно для
старта и первых десятков агентов.

**ClickHouse — поздняя стадия (фаза 8).** Когда объём записей начнёт перерастать
PostgreSQL (миллионы проверок, тяжёлые агрегации и latency percentiles), метрики
переезжают в ClickHouse. Слой доступа к метрикам изолируется за интерфейсом, чтобы
переезд не затронул остальной код.

Пример записи:

```json
{
  "monitor_id": "mon_123",
  "agent_id": "agent_de_01",
  "timestamp": "2026-07-11T02:30:00Z",
  "status": "up",
  "latency_ms": 87,
  "dns_ms": 12, "connect_ms": 24, "tls_ms": 31, "response_ms": 20,
  "status_code": 200
}
```

Идеально для временных рядов, графиков, агрегаций, uptime за период, latency
percentiles и большого числа записей. DDL — в [`../packages/database/`](../packages/database/).

## Redis — кэш и координация

Кэш, rate limiting, блокировки, временные состояния, debounce уведомлений,
realtime presence агентов, сессии, короткоживущие токены.

> Redis **не** должен быть основной очередью мониторинга — для этого NATS.

## NATS JetStream — шина задач и событий

Основная шина. Durable consumers, acknowledgements, повторная доставка,
wildcard subjects.

События:

```text
monitor.check.requested   monitor.check.completed   monitor.status.changed
incident.opened           incident.resolved         notification.requested
agent.connected           agent.disconnected
```

Subjects:

```text
checks.eu.de.agent-001    checks.eu.*
events.monitor.result     events.incident.opened
notifications.telegram
```

## Правило доступа

Агенты **не имеют** прямого доступа к PostgreSQL/ClickHouse — только через
Control Plane / NATS. См. [agents-protocol.md](agents-protocol.md).
