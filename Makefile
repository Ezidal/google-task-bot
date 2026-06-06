.PHONY: help deploy deploy-fresh build up down logs restart test run

COMPOSE := docker compose

help:
	@echo "Google Task Bot"
	@echo ""
	@echo "  make deploy        — git pull + пересборка и перезапуск (Docker)"
	@echo "  make deploy-fresh  — то же, но без кэша Docker"
	@echo "  make build         — собрать образ"
	@echo "  make up            — запустить контейнер"
	@echo "  make down          — остановить контейнер"
	@echo "  make restart       — перезапустить без пересборки"
	@echo "  make logs          — логи (tail -f)"
	@echo "  make test          — go test ./..."
	@echo "  make run           — локальный запуск без Docker"

deploy:
	git pull
	$(COMPOSE) up -d --build

deploy-fresh:
	git pull
	$(COMPOSE) build --no-cache
	$(COMPOSE) up -d

build:
	$(COMPOSE) build

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart:
	$(COMPOSE) restart

logs:
	$(COMPOSE) logs -f --tail=50

test:
	go test ./...

run:
	go run ./cmd/bot
