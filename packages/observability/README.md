# observability

Общий код наблюдаемости: инициализация OpenTelemetry (traces/metrics), structured
logging (slog), экспорт в Prometheus/Loki/Tempo. Ключевая метрика платформы —
`scheduler_lag_seconds`.

См. [../../docs/architecture.md](../../docs/architecture.md), ROADMAP фаза 7.
