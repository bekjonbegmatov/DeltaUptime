# Протокол probe-агентов

## Принцип связи

Агент устанавливает **исходящее постоянное соединение** — не открывает портов,
работает за NAT:

- **Рекомендуется:** gRPC stream over TLS **или** NATS over TLS.
- **Для первой версии допустимо:** polling `GET /agent/tasks` + `POST /agent/results`.

Постоянное соединение даёт: мгновенную отправку заданий, heartbeat, online/offline,
обновление конфигурации без polling.

## Жизненный цикл

```text
1. Админ создаёт агента → CP выдаёт одноразовый enrollment token.
2. Агент шлёт token + системную информацию.
3. CP выдаёт agent ID + постоянные credentials.
4. Агент устанавливает постоянное исходящее соединение.
5. Агент: heartbeat + приём заданий + отправка результатов.
```

## Регистрация

```json
{
  "hostname": "de-frankfurt-01",
  "version": "1.0.0",
  "region": "de-frankfurt",
  "public_ip": "203.0.113.10",
  "capabilities": ["http", "tcp", "icmp", "dns", "tls"]
}
```

## Задание проверки (CP → агент)

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

`check_id` уникален → идемпотентность: повтор одного и того же задания не создаёт
дублей результатов.

## Результат (агент → CP)

```json
{
  "check_id": "chk_123",
  "monitor_id": "mon_123",
  "agent_id": "agent_de_01",
  "timestamp": "2026-07-11T02:30:00Z",
  "status": "up",
  "latency_ms": 87,
  "dns_ms": 12, "connect_ms": 24, "tls_ms": 31, "response_ms": 20,
  "status_code": 200
}
```

## Безопасность

- TLS обязателен; mTLS или короткоживущие credentials с ротацией.
- Агент соблюдает SSRF-политику от CP (см. [security.md](security.md)).
- Контракты сообщений живут в [`../packages/protocol/`](../packages/protocol/).
