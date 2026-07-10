# Публичные Status Pages

Status Page — отдельный продукт внутри системы, отдельное приложение
[`apps/status-web`](../apps/status-web/) на своём домене (`status.domain.com` или
пользовательский). Не путать с приватной панелью `apps/panel-web`.

## Компоненты (группировка мониторов)

Пользователь показывает не каждый монитор, а логические компоненты:

```text
Delta VPN
├── API           (Main API, Auth API)
├── VPN Servers   (Germany, Netherlands, Finland)
├── Telegram Bot
└── Payments
```

Статус компонента рассчитывается:

```text
All healthy       → Operational
One unavailable   → Partial Outage
Majority down     → Major Outage
High latency      → Degraded
Maintenance       → Maintenance
```

## Конструктор

- **Простой режим:** цвета, шрифт, радиусы, логотип, тема → генерируются CSS-переменные.
- **Расширенный режим:** Custom CSS с realtime preview (sandbox). Ограничения
  безопасности — [security.md](security.md).
- **Drag-and-drop layout** блоков (header, global status, components, uptime history,
  incidents, subscribe form, footer), сохраняется как JSON.

```json
{
  "layout": [
    { "type": "header", "enabled": true },
    { "type": "global_status", "enabled": true },
    { "type": "components", "enabled": true, "columns": 2 },
    { "type": "incidents", "enabled": true, "limit": 10 }
  ]
}
```

## Функции

Собственный домен, логотип/favicon/цвета, светлая/тёмная тема, история инцидентов,
planned maintenance, uptime за 90 дней, подписка (email/webhook/RSS/Telegram-канал),
готовые темы, white-label.

## White-label по тарифам

```text
Free:     поддомен, брендинг платформы, базовые цвета
Pro:      свой домен, Custom CSS, свой логотип
Business: white-label, несколько status pages, свои email-шаблоны, расширенные роли
```

## Отказоустойчивость

status-web отдаётся с CDN-кэшированием и SSR. Если основная панель упадёт —
публичные страницы продолжают отдаваться из кэша.

---

## Палитра панели (dark, в стиле спецификации)

```css
:root {
  --page-bg:       #10161e;
  --sidebar:       #111922;
  --card-bg:       #171f29;
  --border:        #2b3542;
  --primary:       #31cee2;  /* бирюзовый акцент */
  --status-up:     #34d399;
  --status-warning:#f59e0b;
  --status-down:   #ef5350;
  --text-primary:  #f4f7fb;
  --text-secondary:#8995a5;
  --border-radius: 12px;
}
```

Активный пункт меню — с левым/правым бирюзовым бордером и лёгкой подсветкой фона.
