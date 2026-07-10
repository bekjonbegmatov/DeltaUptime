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
    ├── auth/                # JWT/TOTP/WebAuthn/api-keys/RBAC/audit для identity-фазы
    ├── config/              # загрузка конфигурации из env
    └── httpapi/             # HTTP-сервер: /healthz, /readyz, graceful shutdown
```

Дальше сюда добавляются users, organizations, monitors, incidents и прочие модули;
роутер, вероятно, позже переедет на Chi.

## Локально

```bash
make build && ./bin/uptime-server api    # или: make run-api
./bin/uptime-server version
make check                               # go vet + go test (перед коммитом)
```

Сейчас уже есть identity-слой поверх PostgreSQL/sqlc:

- `/v1/auth/register|login|refresh|me`
- `/v1/auth/totp/setup|enable|disable`
- `/v1/auth/webauthn/register/begin|finish`
- `/v1/auth/webauthn/login/begin|finish`
- `/v1/organizations/...` для membership/API-key/audit управления

`scheduler` и `worker` пока остаются заглушками. См. [../../ROADMAP.md](../../ROADMAP.md)
фазы 0–1.

Общая архитектура — [../../docs/architecture.md](../../docs/architecture.md),
правила — [../../AGENTS.md](../../AGENTS.md).
