# Scheduler — планировщик проверок

Один из важнейших компонентов. **Scheduler сам не выполняет проверки.** Он решает:
какой монитор, когда, каким агентом, из какого региона должен быть проверен, и
кладёт задание в NATS.

## Пример конфигурации монитора

```text
Monitor: example.com
Interval: 30 секунд
Regions: Germany, Netherlands, Russia
Required confirmations: 2 из 3
```

## Задание

```json
{
  "check_id": "chk_123",
  "monitor_id": "mon_123",
  "type": "http",
  "target": "https://example.com/health",
  "timeout_ms": 5000,
  "scheduled_at": "2026-07-11T02:30:00Z"
}
```

## Идемпотентность и отсутствие дублей

Чтобы несколько инстансов scheduler не создавали задания дважды:

- **leader election** / distributed lock;
- **PostgreSQL advisory locks** — простой способ запустить несколько инстансов;
- уникальный `check_id`;
- идемпотентность на стороне приёма результатов.

## Ключевая метрика

```text
scheduler_lag_seconds
```

Если проверка должна была запуститься в 02:10:00, а запустилась в 02:10:15 —
scheduler lag = 15 сек. За этой метрикой следим в Grafana (см. ROADMAP, фаза 7).
