# uptime-agent — probe-агент DeltaUptime

Probe-агент (сетевой зонд) — статически собранный Go-бинарь, который выполняет
проверки (HTTP/TCP/ICMP/DNS/TLS) из своей точки присутствия (регион, оператор,
дата-центр) и отправляет результаты в Control Plane.

> Это **сетевой** агент платформы. Не путать с AI-агентами разработки
> в [`.claude/agents/`](../../.claude/agents/).

---

## Принципы

- **Один статический бинарь**, никаких зависимостей рантайма.
- **Нет прямого доступа к БД.** Только `Agent → Control Plane / NATS`.
- **Только исходящее соединение** (работает за NAT, без открытых портов):
  gRPC stream over TLS или NATS over TLS.
- Агент получает задания, выполняет, шлёт результаты + heartbeat.

```text
Control Plane ──tasks──▶  ┌─────────────┐
                          │ uptime-agent │  HTTP / TCP / ICMP / DNS / TLS
Control Plane ◀─results─  └─────────────┘
Control Plane ◀heartbeat─
```

Полный протокол и жизненный цикл — [../../docs/agents-protocol.md](../../docs/agents-protocol.md).

---

## Регистрация (enrollment)

```text
1. Администратор создаёт агента в панели.
2. Control Plane выдаёт одноразовый enrollment token.
3. Агент отправляет token + системную информацию.
4. Control Plane выдаёт agent ID и постоянные credentials.
5. Агент устанавливает постоянное исходящее соединение.
```

Полезная нагрузка при регистрации:

```json
{
  "hostname": "de-frankfurt-01",
  "version": "1.0.0",
  "region": "de-frankfurt",
  "public_ip": "203.0.113.10",
  "capabilities": ["http", "tcp", "icmp", "dns", "tls"]
}
```

---

## Запуск

### Бинарь

```bash
./uptime-agent \
  --server=https://panel.example.com \
  --token=agent_token \
  --region=de-frankfurt
```

### Переменные окружения

| Переменная | Описание |
|------------|----------|
| `CONTROL_PLANE_URL` | URL Control Plane |
| `AGENT_TOKEN` | enrollment или постоянный токен |
| `AGENT_REGION` | код региона (`de-frankfurt`, `ru-msk`…) |
| `AGENT_LABELS` | доп. метки: `provider=hetzner,network=datacenter` |

### Docker

```yaml
services:
  uptime-agent:
    image: registry.example.com/uptime-agent:latest
    restart: unless-stopped
    network_mode: host          # нужен для ICMP и корректных таймингов
    environment:
      CONTROL_PLANE_URL: https://uptime.example.com
      AGENT_TOKEN: secret
      AGENT_REGION: de-frankfurt
```

### systemd

См. [../../deployments/systemd/](../../deployments/systemd/).

---

## Что измеряет агент

Для HTTP-проверки возвращается полная разбивка таймингов:

```text
DNS lookup → TCP connect → TLS handshake → TTFB → Total response time
+ HTTP status, response size, error category
```

Пример результата, отправляемого в Control Plane:

```json
{
  "monitor_id": "mon_123",
  "agent_id": "agent_de_01",
  "timestamp": "2026-07-11T02:30:00Z",
  "status": "up",
  "latency_ms": 87,
  "dns_ms": 12,
  "connect_ms": 24,
  "tls_ms": 31,
  "response_ms": 20,
  "status_code": 200
}
```

Типы проверок и их параметры — [../../docs/monitors.md](../../docs/monitors.md).

---

## Безопасность агента

- Все соединения по TLS; mTLS или короткоживущие credentials с ротацией.
- Агент **обязан** соблюдать SSRF-ограничения, приходящие от Control Plane:
  для публичных мониторов запрещены приватные диапазоны и metadata-эндпоинты.
- Для private-агентов (инфраструктура заказчика) ограничения настраиваются отдельно.

Детали — [../../docs/security.md](../../docs/security.md).

---

## Разработка

```bash
cd apps/agent
go build ./...       # сборка
go test ./...        # тесты (обязательно перед коммитом)
```

Правила коммитов и git — [../../AGENTS.md](../../AGENTS.md).
Контракты сообщений — в [`packages/protocol`](../../packages/protocol/).
