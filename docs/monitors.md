# Типы мониторов

## HTTP / HTTPS

Настройки: URL, метод, headers, request body, timeout, redirects, разрешённые
status codes, проверка текста/regex/JSONPath/keyword, макс. время ответа, TLS
validation, IPv4/IPv6, конкретный IP при заданном Host.

Метрики (полная разбивка таймингов):

```text
DNS lookup → TCP connect → TLS handshake → TTFB → Total response time
+ HTTP status, response size
```

## TCP

host, port, connect timeout, optional send payload, expected response, TLS mode.

## ICMP

packet loss, min/max/avg latency, jitter.

## DNS

resolver, record type, expected answer, DNSSEC, response time.

## TLS certificate

срок действия, issuer, SAN, hostname validation, chain validity.
Предупреждения за **30 / 14 / 7 / 3** дня до истечения.

## Позже

UDP, WebSocket, gRPC, SMTP, IMAP, PostgreSQL, MySQL, Redis, SSH banner, Minecraft,
custom script, browser check (Playwright).

---

## Модель определения DOWN (state machine)

Один timeout — ещё не инцидент.

```text
UP → DEGRADED → PENDING_DOWN → DOWN → PENDING_UP → UP
```

Пример политики:

```text
1 ошибка          → PENDING_DOWN
3 ошибки подряд   → DOWN
2 успеха подряд   → UP
```

## Кворум для распределённых проверок

DOWN засчитывается только при подтверждении несколькими регионами:

```text
DOWN, если ошибку подтвердили минимум 2 региона
  ИЛИ quorum = 60% агентов

Пример: 5 агентов, 3 подтверждают ошибку → monitor DOWN
```

Это защищает от ложных тревог, когда проблема только у одного агента/региона.
Пример вывода по регионам (видно, что упал не сервис, а один регион):

```text
Germany      200 OK    52 ms
Netherlands  200 OK    61 ms
Russia       Timeout   5000 ms
Finland      200 OK    72 ms
```
