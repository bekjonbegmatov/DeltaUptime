# Безопасность и anti-abuse

## Обязательный минимум

- TLS для всех соединений; mTLS или короткоживущие agent credentials + ротация.
- Argon2id для паролей; TOTP/WebAuthn; refresh-token rotation.
- Audit log; permission-based RBAC; IP allowlist для системной админки.
- Rate limits; API scopes; encrypted secrets.

## SSRF — критично

Пользователь создаёт HTTP-мониторы → через монитор можно атаковать внутреннюю сеть
агента. Для **обычных** пользователей запрещаем проверять:

```text
127.0.0.0/8      10.0.0.0/8      172.16.0.0/12
192.168.0.0/16   169.254.0.0/16  localhost
metadata endpoints (169.254.169.254 и аналоги)
```

Дополнительно: проверка **DNS rebinding**, ограничение redirect, ограничение
response size. Для private-агентов (инфраструктура заказчика) ограничения
настраиваются отдельно.

## Anti-abuse

Открытая система может быть превращена в сканер портов или DDoS-инструмент.
Необходимы:

- минимальный интервал проверки; ограничение timeout;
- ограничение размера request body и числа headers;
- лимиты мониторов и регионов; запрет диапазонов IP;
- ограничение redirect и response size; ограничение custom scripts;
- audit всех изменений; автоблокировка подозрительных аккаунтов.

Минимальные интервалы по тарифам:

```text
Free: 60s   Pro: 30s   Business: 10s   Private infra: 5s
```

## Status pages Custom CSS

Custom CSS разрешаем только на публичном `status-web` (отдельный домен), с
санитизацией, запретом `@import`/внешних URL, лимитом размера, версионированием и
кнопкой сброса, preview в sandbox iframe. Произвольный JavaScript в v1 **запрещён**
(только Analytics ID и sanitized HTML-footer). Детали — [status-pages.md](status-pages.md).
