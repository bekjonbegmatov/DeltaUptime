# DeltaUptime

[![CI](https://github.com/bekjonbegmatov/DeltaUptime/actions/workflows/ci.yml/badge.svg)](https://github.com/bekjonbegmatov/DeltaUptime/actions/workflows/ci.yml)

> **Uptime Control Plane + Probe Network** — централизованная платформа управления
> распределённой сетью мониторинга. Один ресурс проверяется одновременно из разных
> стран, операторов связи и пользовательских агентов.

DeltaUptime — это не «сайт с пингами», а полноценная distributed uptime-платформа
в духе Remnawave: единый Control Plane, распределённые probe-агенты, мультитенантные
организации и роли, realtime-статусы, инциденты, Telegram-уведомления, публичные
status pages и API для интеграций.

Главное преимущество перед Uptime Kuma и обычными uptime-сервисами —
**распределённость**: проверка идёт из нескольких регионов и сетей, а падением
считается только то, что подтверждено кворумом агентов.

---

## Содержание

- [Архитектура](#архитектура)
- [Технологический стек](#технологический-стек)
- [Структура репозитория](#структура-репозитория)
- [Быстрый старт](#быстрый-старт)
- [Компоненты](#компоненты)
- [Документация](#документация)
- [Разработка и git-процесс](#разработка-и-git-процесс)
- [Дорожная карта](#дорожная-карта)

---

## Архитектура

```text
                         ┌──────────────────────────┐
                         │        Web Panel         │
                         │  Admin / User / Status   │
                         └────────────┬─────────────┘
                                      │ HTTPS
                                      ▼
┌───────────────────────────────────────────────────────────┐
│                       CONTROL PLANE                        │
│  API Gateway                                               │
│       ├── Auth / Users / Organizations                     │
│       ├── Monitors / Agent Groups                          │
│       ├── Scheduler                                        │
│       ├── Incident Engine                                  │
│       ├── Notification Engine                              │
│       ├── Status Page Service                              │
│       └── Realtime Gateway                                 │
└───────────────┬─────────────────┬─────────────────────────┘
                │ Tasks            │ Results / Events
                ▼                  ▼
        ┌────────────────────────────────┐
        │   Message Broker (NATS JS)     │
        └───────────────┬────────────────┘
         ┌──────────────┼───────────────┐
         ▼              ▼               ▼
   ┌───────────┐  ┌───────────┐  ┌───────────┐
   │ Probe RU  │  │ Probe DE  │  │ Probe FI  │   HTTP / TCP / DNS
   │           │  │           │  │           │   ICMP / TLS
   └───────────┘  └───────────┘  └───────────┘
                        │
                        ▼
                ┌────────────────┐
                │  ClickHouse    │  метрики проверок
                └────────────────┘
```

**Ключевое архитектурное решение:** начинаем с **модульного монолита** Control Plane
+ отдельные агенты + отдельные worker-процессы. Не 15 микросервисов сразу.
Когда нагрузка вырастет — `scheduler`, `notifications` и `incident engine`
выносятся в отдельные сервисы. Подробнее: [docs/architecture.md](docs/architecture.md).

Один бинарь Go, разные процессы:

```bash
uptime-server migrate     # применить миграции
uptime-server api         # HTTP API + realtime gateway
uptime-server scheduler   # планировщик проверок
uptime-server worker      # incident + notification workers
```

---

## Технологический стек

| Слой | Технология |
|------|-----------|
| Backend | Go + Chi + pgx + sqlc |
| Frontend (панель) | Next.js + TypeScript + Tailwind + shadcn/ui + TanStack Query/Table + ECharts |
| Frontend (status) | Отдельный Next.js-проект, SSR + CDN caching |
| Probe-агент | Go (один статический бинарь) |
| Источник истины | PostgreSQL |
| Метрики проверок | PostgreSQL на старте → **ClickHouse при росте** (поздняя стадия) |
| Message broker | NATS JetStream |
| Cache / locks | Redis |
| Realtime | SSE |
| Auth | Argon2id + JWT access + rotating refresh + TOTP/WebAuthn |
| Observability | structured logs + OpenTelemetry-ready; **Prometheus + Grafana — поздняя стадия** |
| Deploy | **Docker / Docker Compose** (Kubernetes не используем) |
| CI/CD | GitHub Actions |

> **Стадийность инфраструктуры.** Базовый стек намеренно минимален: PostgreSQL +
> Redis + NATS в Docker Compose. **ClickHouse, Prometheus и Grafana — поздняя
> стадия** (вводятся, когда объём метрик и потребность в дашбордах реально
> вырастут). **Kubernetes в проекте не используется** — только Docker.

Подробнее по выбору стека и библиотек: [docs/architecture.md](docs/architecture.md).

---

## Структура репозитория

```text
DeltaUptime/
├── apps/
│   ├── control-plane/   # API Gateway + модули (auth, users, orgs, monitors…)
│   ├── scheduler/       # планировщик проверок (leader election)
│   ├── worker/          # incident engine + notification workers
│   ├── agent/           # probe-агент (см. apps/agent/README.md)
│   ├── panel-web/       # приватная админ-панель (panel.domain.com)
│   └── status-web/      # публичные status pages (status.domain.com)
├── packages/
│   ├── protocol/        # gRPC/NATS-контракты между CP и агентами
│   ├── database/        # схемы, sqlc-запросы, ClickHouse DDL
│   ├── observability/   # общий OTel/logging/metrics-код
│   └── shared/          # общие типы и утилиты
├── deployments/
│   ├── docker-compose/  # локальный и prod-lite стек
│   ├── kubernetes/      # манифесты для роста
│   └── systemd/         # unit-файлы для агентов на VPS
├── migrations/          # SQL-миграции (Goose)
├── docs/                # архитектура, БД, протокол, безопасность…
├── scripts/             # dev/build/release-скрипты
├── .claude/agents/      # инструкции для AI-агентов разработки
├── README.md
├── ROADMAP.md           # дорожная карта по фазам
├── PROGRESS.md          # журнал прогресса (обновляется по мере работы)
└── AGENTS.md            # правила для разработчиков и AI-агентов (git, тесты, коммиты)
```

> ⚠️ Не путать **probe-агенты** (`apps/agent/` — сетевые зонды платформы) и
> **AI-агенты разработки** (`.claude/agents/` — инструкции для Claude Code).

---

## Быстрый старт

Требования: Docker + Docker Compose, Go 1.23+, Node 20+.

```bash
# 1. Поднять базовую инфраструктуру (postgres, redis, nats)
cd deployments/docker-compose
docker compose up -d

# 2. Применить миграции
uptime-server migrate

# 3. Запустить процессы Control Plane (в отдельных терминалах)
uptime-server api
uptime-server scheduler
uptime-server worker

# 4. Запустить панель
cd apps/panel-web && npm install && npm run dev
```

Запуск probe-агента — см. [apps/agent/README.md](apps/agent/README.md).

---

## Компоненты

| Компонент | Описание | Документация |
|-----------|----------|--------------|
| Control Plane | Единый backend, модульный монолит | [docs/architecture.md](docs/architecture.md) |
| Scheduler | Кто/когда/каким агентом проверяет | [docs/scheduler.md](docs/scheduler.md) |
| Probe-агент | Сетевой зонд (HTTP/TCP/ICMP/DNS/TLS) | [apps/agent/README.md](apps/agent/README.md) |
| Incident Engine | State machine + кворум DOWN | [docs/incidents.md](docs/incidents.md) |
| Notifications | Telegram/webhook/email + эскалации | [docs/notifications.md](docs/notifications.md) |
| Status Pages | Публичные страницы + конструктор | [docs/status-pages.md](docs/status-pages.md) |
| Мониторы | Типы проверок и метрики | [docs/monitors.md](docs/monitors.md) |
| Безопасность | SSRF, RBAC, anti-abuse | [docs/security.md](docs/security.md) |
| Данные | PostgreSQL + Redis + NATS (ClickHouse — поздняя стадия) | [docs/database.md](docs/database.md) |

---

## Документация

Вся документация — в [docs/](docs/). Начните с
[docs/architecture.md](docs/architecture.md), затем [docs/database.md](docs/database.md)
и [docs/agents-protocol.md](docs/agents-protocol.md).

---

## Разработка и git-процесс

Правила для контрибьюторов и AI-агентов — в [AGENTS.md](AGENTS.md).

Кратко:

1. **Никаких коммитов без тестов.** Коммит делается только после того, как код
   собирается и проходит нормальное тестирование.
2. Работа ведётся в feature-ветках, не в `main`.
3. Формат коммитов — **Conventional Commits** (`feat:`, `fix:`, `docs:`, `chore:`…).
4. Прогресс фиксируется в [PROGRESS.md](PROGRESS.md).

---

## Дорожная карта

Полный план по фазам — в [ROADMAP.md](ROADMAP.md). Цель первой версии — не урезанный
учебный MVP, а полноценный первый релиз с мультитенантностью, распределёнными
проверками, кворум-инцидентами, Telegram и публичными status pages.

---

## Лицензия

TBD (кандидаты: AGPL-3.0 или Apache-2.0 — определить перед публичным релизом).
