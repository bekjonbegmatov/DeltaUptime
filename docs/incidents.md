# Incident Engine

Инцидент — **отдельная сущность**, а не строка с ошибкой.

```text
Incident:
  id, organization, monitor
  started_at, acknowledged_at, resolved_at
  severity, current_state
  affected_regions, failure_reason
  notification_status
```

## Пример таймлайна

```text
02:10:01  Германия: timeout
02:10:04  Нидерланды: timeout
02:10:05  Инцидент открыт
02:10:10  Telegram отправлен
02:14:20  Германия: success
02:14:22  Нидерланды: success
02:14:22  Инцидент закрыт
```

## Возможности

- acknowledge, silence, maintenance mode
- manual resolution, incident comments, postmortem
- escalation

State machine определения UP/DOWN и кворум — в [monitors.md](monitors.md).
Уведомления по событиям инцидента — в [notifications.md](notifications.md).
