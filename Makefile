# DeltaUptime — dev tasks. Требования: Go 1.25+, Docker.

BIN      := bin/uptime-server
PKG      := ./apps/control-plane/cmd/uptime-server
COMPOSE  := docker compose -f deployments/docker-compose/docker-compose.yml

.PHONY: help build test vet lint check run-api up down ps clean

help: ## Показать список команд
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  %-12s %s\n", $$1, $$2}'

build: ## Собрать бинарь uptime-server
	go build -o $(BIN) $(PKG)

test: ## Прогнать unit-тесты
	go test ./...

vet: ## go vet
	go vet ./...

lint: ## golangci-lint (если установлен)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || \
		echo "golangci-lint не установлен — пропущено"

check: vet test ## Проверки перед коммитом (ЗЕЛЁНО обязательно)

run-api: build ## Запустить HTTP API локально
	$(BIN) api

up: ## Поднять базовую инфраструктуру (postgres+redis+nats)
	$(COMPOSE) up -d

down: ## Остановить инфраструктуру
	$(COMPOSE) down

ps: ## Статус контейнеров
	$(COMPOSE) ps

clean: ## Удалить артефакты сборки
	rm -rf bin
