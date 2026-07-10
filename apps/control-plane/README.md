# control-plane

Модульный монолит Control Plane (Go). API Gateway + модули: auth, users,
organizations, agents, monitors, scheduler, checks, incidents, notifications,
status-pages, billing, audit, integrations.

Собирается в бинарь `uptime-server`; подкоманды `api|scheduler|worker|migrate|version`.

## Структура кода

```text
control-plane/
├── cmd/uptime-server/       # main: подхват сигналов + dispatch
└── internal/
    ├── app/                 # разбор подкоманд (тестируемо, не в main)
    ├── config/              # загрузка конфигурации из env
    └── httpapi/             # HTTP-сервер: /healthz, /readyz, graceful shutdown
```

Дальше сюда добавляются модули (auth, users, organizations, monitors, incidents…),
роутер, вероятно, переедет на Chi, появится доступ к БД через pgx + sqlc.

## Локально

```bash
make build && ./bin/uptime-server api    # или: make run-api
./bin/uptime-server version
make check                               # go vet + go test (перед коммитом)
```

Текущий статус — только stdlib (без внешних зависимостей); `migrate`, `scheduler`,
`worker` — заглушки. См. [../../ROADMAP.md](../../ROADMAP.md) фаза 0.

Общая архитектура — [../../docs/architecture.md](../../docs/architecture.md),
правила — [../../AGENTS.md](../../AGENTS.md).
