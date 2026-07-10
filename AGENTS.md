# AGENTS.md — правила разработки

Документ для людей и для AI-агентов (Claude Code и др.), работающих над DeltaUptime.
Читается **перед** любыми изменениями в репозитории.

---

## 1. Golden rules

1. **Не коммить непротестированный код.** Коммит допустим только после того, как:
   - код собирается (`go build ./...` / `npm run build`);
   - проходят соответствующие тесты (`go test ./...` / `npm test`);
   - линтер чист (`golangci-lint run` / `npm run lint`).
2. **Никогда не работать напрямую в `main`.** Всегда feature-ветка.
3. **Один PR — одна логическая задача.** Не смешивать рефакторинг и фичу.
4. **Секреты не коммитятся.** Только `.env.example`, реальные `.env` — в `.gitignore`.
5. **Обновляй [PROGRESS.md](PROGRESS.md)** после каждого завершённого блока работы.

---

## 2. Инфраструктура проекта

Модульный монолит + отдельные процессы. Один Go-бинарь, разные подкоманды:

```bash
uptime-server migrate     # применить миграции (Goose)
uptime-server api         # HTTP API + realtime (SSE)
uptime-server scheduler   # планировщик (PostgreSQL advisory locks / leader election)
uptime-server worker      # incident engine + notification workers
```

Хранилища и их роли (подробно — [docs/database.md](docs/database.md)):

| Хранилище | Роль | Что нельзя |
|-----------|------|-----------|
| PostgreSQL | Источник истины + **результаты проверок на старте** | — |
| Redis | Кэш, rate-limit, locks, presence, debounce | Быть основной очередью |
| NATS JetStream | Шина задач и событий | Хранить состояние как в БД |
| ClickHouse | Метрики проверок **при росте** (поздняя стадия) | Вводить раньше времени |

> **Стадийность.** Базовый стек — только PostgreSQL + Redis + NATS в Docker.
> **ClickHouse, Prometheus, Grafana — поздняя стадия.** **Kubernetes не используется**
> — деплой только через Docker / Docker Compose.

**Агенты не имеют прямого доступа к PostgreSQL/ClickHouse.** Только
`Agent → Control Plane / NATS`. См. [docs/agents-protocol.md](docs/agents-protocol.md).

---

## 3. Git-процесс

### 3.1 Ветки

```text
main                 # всегда зелёный, защищён
  └── feat/<scope>-<short-desc>     # новая функциональность
  └── fix/<scope>-<short-desc>      # исправление
  └── docs/<short-desc>             # документация
  └── chore/<short-desc>            # инфраструктура, зависимости
```

Пример: `feat/monitors-http-check`, `fix/scheduler-double-dispatch`.

### 3.2 Порядок работы

```bash
git switch -c feat/monitors-http-check   # ветка от свежего main
# ... код ...
go build ./... && go test ./... && golangci-lint run   # ЗЕЛЁНО перед коммитом
git add -p                                # осознанный стейджинг
git commit                                # Conventional Commit (см. ниже)
git push -u origin feat/monitors-http-check
gh pr create                              # PR в main
```

### 3.3 Формат коммитов — Conventional Commits

```text
<type>(<scope>): <краткое описание в императиве>

[тело: что и зачем, не «как»]

[footer: Closes #123, BREAKING CHANGE: …]
```

Типы: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `build`, `ci`.

Scope — модуль: `auth`, `monitors`, `scheduler`, `incidents`, `notifications`,
`status-pages`, `agent`, `panel-web`, `db`, `deploy`.

Примеры:

```text
feat(monitors): add HTTP/HTTPS check with DNS/TCP/TLS timing breakdown
fix(scheduler): prevent duplicate dispatch via advisory lock
docs(architecture): describe quorum-based DOWN detection
chore(deploy): add clickhouse to docker-compose stack
```

### 3.4 Когда делать первый коммит

Пока в репозитории только скелет и документация — их можно закоммитить как
`chore: scaffold project structure and docs`. **Реальный код фич коммитится
только после прохождения тестов** (см. Golden rule №1).

---

## 4. Тестирование (обязательно перед коммитом)

| Область | Команда | Критерий |
|---------|---------|----------|
| Go build | `go build ./...` | без ошибок |
| Go unit | `go test ./...` | все зелёные |
| Go lint | `golangci-lint run` | чисто |
| Frontend build | `npm run build` | без ошибок |
| Frontend lint | `npm run lint` | чисто |
| Integration | `docker compose -f deployments/docker-compose/docker-compose.test.yml up --abort-on-container-exit` | pass |

Для критичной логики (state machine инцидентов, кворум DOWN, scheduler
идемпотентность) — обязательны unit-тесты на граничные случаи.

---

## 5. Definition of Done

Задача считается завершённой, когда:

- [ ] Код собирается и проходит тесты/линт.
- [ ] Добавлены тесты на новую логику.
- [ ] Обновлена документация в `docs/` (если менялось поведение/контракт).
- [ ] Обновлён [PROGRESS.md](PROGRESS.md).
- [ ] Обновлён [ROADMAP.md](ROADMAP.md) (отмечен закрытый пункт).
- [ ] Нет закоммиченных секретов.
- [ ] Создан PR с осмысленным описанием.

---

## 6. Безопасность (не забывать при разработке)

Пользователь создаёт HTTP-мониторы → **риск SSRF**. Обязательно блокировать для
обычных пользователей приватные диапазоны и metadata-эндпоинты, проверять DNS
rebinding, ограничивать redirect и response size. Полный чеклист —
[docs/security.md](docs/security.md). Anti-abuse (минимальные интервалы, лимиты) —
там же.

---

## 7. AI-агенты разработки

Инструкции для специализированных AI-агентов — в [.claude/agents/](.claude/agents/).
Каждый агент отвечает за свой домен (backend, frontend, probe-агент, инфраструктура,
безопасность). Перед задачей агент читает этот файл и профильную доку в `docs/`.
