# status-web

Публичные status pages (`status.domain.com` и пользовательские домены). Отдельное
Next.js-приложение: SSR + CDN caching, конструктор (темы, drag-and-drop layout,
Custom CSS в sandbox), подписки, white-label.

Изолировано от админ-панели: без доступа к админ-cookie, отдаётся из кэша при
падении Control Plane. Док: [../../docs/status-pages.md](../../docs/status-pages.md).
