# control-plane

Модульный монолит Control Plane (Go). API Gateway + модули: auth, users,
organizations, agents, monitors, scheduler, checks, incidents, notifications,
status-pages, billing, audit, integrations.

Собирается в бинарь `uptime-server`; подкоманды `api|scheduler|worker|migrate`.

См. [../../docs/architecture.md](../../docs/architecture.md) и
[../../AGENTS.md](../../AGENTS.md). Статус — фаза 0/1 в [../../ROADMAP.md](../../ROADMAP.md).
