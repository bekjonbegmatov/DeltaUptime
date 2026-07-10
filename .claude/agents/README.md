# AI-агенты разработки

Определения специализированных субагентов Claude Code для работы над DeltaUptime.
Каждый файл — один агент (frontmatter + инструкции). Не путать с probe-агентами
платформы ([`apps/agent/`](../../apps/agent/)).

Перед любой задачей агент читает [`AGENTS.md`](../../AGENTS.md) (git/тесты/коммиты)
и профильную доку в [`docs/`](../../docs/).

| Агент | Домен | Файл |
|-------|-------|------|
| backend | Control Plane, scheduler, worker (Go) | [backend.md](backend.md) |
| frontend | panel-web / status-web (Next.js) | [frontend.md](frontend.md) |
| probe-agent | сетевой probe-агент (Go) | [probe-agent.md](probe-agent.md) |
| infra | Docker/K8s/CI/observability | [infra.md](infra.md) |
| security | SSRF, RBAC, anti-abuse, аудит | [security.md](security.md) |

Общее правило для всех: **не коммитить непротестированный код**, работать в
feature-ветке, Conventional Commits, обновлять `PROGRESS.md` и `ROADMAP.md`.
