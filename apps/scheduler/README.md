# scheduler

Планировщик проверок. Решает кто/когда/каким агентом/из какого региона проверяется,
кладёт задания в NATS. Идемпотентность через advisory locks и уникальный check_id.
На старте — часть `uptime-server` (`uptime-server scheduler`), при росте — отдельный
сервис.

Док: [../../docs/scheduler.md](../../docs/scheduler.md).
