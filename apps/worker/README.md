# worker

Фоновые worker-процессы: Incident Engine (state machine + кворум DOWN) и
Notification worker (Telegram/webhook/email, дедупликация, retry, backoff,
escalation). На старте — `uptime-server worker`, при росте — отдельные сервисы.

Доки: [../../docs/incidents.md](../../docs/incidents.md),
[../../docs/notifications.md](../../docs/notifications.md).
