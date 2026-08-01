# Atajos del proyecto FinTrack.
# En Windows usa el equivalente: .\make.ps1 <target>
# (Git Bash traduce las rutas absolutas al pasarlas a docker y rompe "make seed";
#  por eso en Windows se usa make.ps1 y este Makefile queda para Linux, macOS y el CI.)
#
# Algunos targets apuntan a codigo que aun no existe (backend, frontend, semilla):
# empiezan a funcionar en cuanto aterriza su fase. Ver el README para el estado.

COMPOSE_DEV = docker compose -f docker-compose.dev.yml
MONGO_URI_DEV = mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin
MONGOSH = mongosh -u fintrack_admin -p fintrack_dev_2026 --authenticationDatabase admin --quiet

.DEFAULT_GOAL := help
.PHONY: help up down dev test test-integracion lint seed build

help: ## Muestra esta ayuda
	@echo "FinTrack - atajos disponibles:"
	@echo "  make up      Levanta MongoDB (compose de desarrollo)"
	@echo "  make down    Detiene MongoDB"
	@echo "  make dev     Levanta Mongo y arranca la API en modo desarrollo   [fase 2]"
	@echo "  make test    Corre las pruebas de Go con cobertura"
	@echo "  make test-integracion  Todas las pruebas, incluidas las de MongoDB"
	@echo "  make lint    Corre go vet y golangci-lint                        [fase 2]"
	@echo "  make seed    Recrea el esquema y carga los datos semilla"
	@echo "  make build   Compila el backend y el frontend                    [fase 7]"

up: ## Levanta MongoDB en segundo plano
	$(COMPOSE_DEV) up -d
	@echo "MongoDB escuchando en localhost:27017"

down: ## Detiene MongoDB (conserva los datos del volumen)
	$(COMPOSE_DEV) down

dev: up ## Levanta Mongo y arranca la API con recarga manual
	cd backend && go run ./cmd/api

test: ## Pruebas del backend (las de integracion se saltan solas)
	cd backend && go test ./... -coverpkg=./... -coverprofile=coverage.out
	@cd backend && go tool cover -func=coverage.out | tail -1

test-integracion: up ## Todas las pruebas, incluidas las que necesitan MongoDB
	cd backend && MONGO_URI_PRUEBAS="$(MONGO_URI_DEV)" go test ./... -coverpkg=./... -coverprofile=coverage.out
	@cd backend && go tool cover -func=coverage.out | tail -1

lint: ## Analisis estatico del backend
	cd backend && go vet ./...
	cd backend && golangci-lint run || echo "golangci-lint no instalado, se omite"

seed: ## Recrea el esquema y carga los datos semilla (idempotente)
	$(COMPOSE_DEV) exec -T mongo $(MONGOSH) --file /scripts/01_crear_colecciones.js
	$(COMPOSE_DEV) exec -T mongo $(MONGOSH) --file /scripts/02_insertar_datos.js

build: ## Compila el binario de la API y el bundle del frontend
	cd backend && go build -o bin/api ./cmd/api
	cd frontend && npm ci && npm run build
