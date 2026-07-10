# Уведомления

Отдельный **notification worker**. Получает события:

```text
incident.opened   incident.resolved   tls.expiring
agent.offline     monitor.degraded
```

## Каналы

Telegram, email, webhook, Discord, Slack, SMS, PagerDuty-совместимый webhook.

## Telegram

Личные сообщения, группы, темы Telegram-групп, inline-кнопки (acknowledge, silence
на 30 минут, открыть монитор/инцидент).

Пример:

```text
🔴 api.example.com недоступен

Проверка: HTTPS
Начало: 02:15:04
Подтверждено: 3 из 4 регионов

🇩🇪 Frankfurt: timeout
🇳🇱 Amsterdam: timeout
🇫🇮 Helsinki: HTTP 502
🇷🇺 Moscow: 200 OK

[Подтвердить] [Отключить на 30 минут]
```

## Обязательно

Дедупликация, retry, exponential backoff, rate limit, grouping, quiet hours,
escalation policies.
