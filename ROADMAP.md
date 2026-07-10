# ROADMAP — DeltaUptime

Дорожная карта по фазам. Цель — не урезанный учебный MVP, а хороший первый релиз,
сравнимый по качеству управления с Remnawave (только вместо VPN-нод — мониторы,
probe-агенты, регионы и инциденты).

Легенда: `[ ]` не начато · `[~]` в работе · `[x]` готово

---

## Фаза 0 — Фундамент (инфраструктура и скелет)

- [x] Структура монорепозитория и документация
- [x] Docker Compose стек — **только базовое:** postgres, redis, nats
      (ClickHouse/Prometheus/Grafana за профилями — поздняя стадия)
- [x] Каркас Go-бинаря `uptime-server` с подкомандами (api/scheduler/worker/migrate)
      — api отдаёт `/healthz` `/readyz`, graceful shutdown, config из env, тесты
- [ ] Миграции (Goose) + sqlc-конфиг (migrate пока заглушка)
- [~] Structured logging (slog) есть; OpenTelemetry-ready интерфейсы — TODO
- [ ] CI (GitHub Actions): build + test + lint

## Фаза 1 — Идентичность и мультитенантность

- [ ] Auth: Argon2id, JWT access + rotating refresh
- [ ] TOTP / WebAuthn
- [ ] Users, Organizations, Memberships
- [ ] Permission-based RBAC (Owner/Admin/Operator/Viewer/Billing)
- [ ] API keys + scopes
- [ ] Audit log

## Фаза 2 — Сеть агентов

- [ ] Регистрация агента (enrollment token → постоянные credentials)
- [ ] Постоянное исходящее соединение (gRPC stream / NATS over TLS)
- [ ] Heartbeat + online/offline presence
- [ ] Agent groups, регионы, провайдеры
- [ ] Версии агентов + rollout

## Фаза 3 — Мониторинг

- [ ] Scheduler (leader election, идемпотентность, advisory locks)
- [ ] HTTP/HTTPS монитор (DNS/TCP/TLS/TTFB timings)
- [ ] TCP монитор
- [ ] ICMP монитор
- [ ] DNS монитор
- [ ] TLS certificate монитор (предупреждения за 30/14/7/3 дня)
- [ ] Распределённые проверки + кворум-определение DOWN
- [ ] Запись метрик проверок в **PostgreSQL** (ClickHouse — поздняя стадия, фаза 8)
- [ ] Maintenance windows

## Фаза 4 — Инциденты и уведомления

- [ ] Incident Engine (state machine: UP→DEGRADED→PENDING_DOWN→DOWN→PENDING_UP→UP)
- [ ] Acknowledge / silence / manual resolve / comments
- [ ] Notification worker: дедупликация, retry, backoff, grouping, quiet hours
- [ ] Telegram (личка/группы/темы, inline-кнопки: ack, silence)
- [ ] Webhook / Email
- [ ] Escalation policies

## Фаза 5 — Панель (panel-web)

- [ ] Dashboard (карточки, uptime 24h, latency, ошибки по регионам)
- [ ] Страница монитора (обзор/результаты/регионы/инциденты/настройки)
- [ ] Страницы агентов, групп, регионов
- [ ] Realtime через SSE
- [ ] Тёмная тема в стиле спецификации (см. docs/status-pages.md — палитра)

## Фаза 6 — Публичные Status Pages (status-web)

- [ ] Отдельное приложение на своём домене
- [ ] Компоненты (группировка мониторов) + расчёт статуса
- [ ] Uptime за 90 дней, история инцидентов, planned maintenance
- [ ] Конструктор: темы, простой редактор + Custom CSS (sanitized, sandbox)
- [ ] Подписки (email/webhook/RSS/Telegram-канал)
- [ ] White-label + собственные домены
- [ ] Отдача из кэша при недоступности основной панели

## Фаза 7 — Системная админка и наблюдаемость (поздняя стадия)

- [ ] System Admin Panel (все орги, лимиты, нагрузка агентов, очереди)
- [ ] Метрики платформы (scheduler_lag_seconds, lost checks, queue depth…)
- [ ] **Prometheus + Grafana** — дашборды и алерты на саму платформу
      (вводятся только на этой стадии, не раньше)

## Фаза 8 — Масштабирование (по необходимости)

- [ ] Вынос scheduler/incident/notification в отдельные сервисы
- [ ] Миграция метрик проверок на **ClickHouse** (когда объём перерастёт PostgreSQL)
- [ ] PostgreSQL HA + replica, ClickHouse cluster, NATS cluster
- [ ] Масштабирование через **Docker** (несколько хостов / compose-профили).
      Kubernetes в проекте **не используется**.

---

## Содержимое «первой правильной версии» (Definition of v1)

1. Мультитенантные организации · 2. Пользователи и роли · 3. Регистрация агентов ·
4. Heartbeat · 5. Agent groups и регионы · 6. HTTP/HTTPS · 7. TCP · 8. ICMP ·
9. TLS cert · 10. Распределённые проверки · 11. Кворум DOWN · 12. Incident engine ·
13. Telegram · 14. История latency · 15. Uptime 24h/7d/30d · 16. Maintenance ·
17. Публичная status page · 18. Audit log · 19. API keys · 20. Системная админка.
