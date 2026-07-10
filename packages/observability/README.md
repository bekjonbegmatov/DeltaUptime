# observability

Общий код наблюдаемости. **На старте:** structured logging (slog) и
OpenTelemetry-ready интерфейсы. **Экспорт в Prometheus/Grafana (+ Loki/Tempo) —
поздняя стадия** (фаза 7), вводится при росте. Ключевая метрика платформы —
`scheduler_lag_seconds`.

См. [../../docs/architecture.md](../../docs/architecture.md), ROADMAP фаза 7.
