SHELL := /bin/bash
.DEFAULT_GOAL := help

# Proyecto de pruebas: volumen y base separados de los de desarrollo y evaluacion.
COMPOSE      := docker compose
COMPOSE_TEST := docker compose -p catalogo-test -f compose.yaml -f compose.test.yaml
COMPOSE_E2E  := docker compose -p catalogo-test -f compose.yaml -f compose.test.yaml -f compose.e2e.yaml
IMAGEN_BUILD := catalogo-build-lint

.PHONY: help setup up down logs migrate test lint check reset-test reset-dev \
        import seed-demo test-persistence evidence e2e _lint-web _lint-go _test _guard _excel

help: ## Lista los objetivos
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-17s %s\n", $$1, $$2}'

setup: ## Crea .env desde .env.example si no existe (si existe, no lo toca)
	@if [ -f .env ]; then echo ".env ya existe, no se toca"; else cp .env.example .env && echo ".env creado desde .env.example"; fi

up: setup ## Construye y levanta la pila (docker compose up --build -d)
	$(COMPOSE) up --build -d

down: ## Apaga la pila sin borrar el volumen
	$(COMPOSE) down

logs: ## Sigue los logs
	$(COMPOSE) logs -f

migrate: setup ## Aplica las migraciones pendientes
	$(COMPOSE) run --rm --build app migrar

test: setup ## Pruebas de Go en contenedor (proyecto catalogo-test, base catalogo_test)
	$(COMPOSE_TEST) run --rm --build pruebas

lint: ## gofmt, go vet y revision de tipos de la interfaz, en contenedor
	@$(MAKE) --no-print-directory _lint-web _lint-go

_lint-web:
	@echo "== tipos de la interfaz (tsc --noEmit, dentro de la etapa web) =="
	docker build --target web -t catalogo-web-lint .

_lint-go:
	@echo "== gofmt y go vet =="
	docker build --target build -t $(IMAGEN_BUILD) .
	docker run --rm $(IMAGEN_BUILD) sh -c 'out=$$(gofmt -l cmd internal tests 2>/dev/null); if [ -n "$$out" ]; then echo "gofmt: archivos sin formato:"; echo "$$out"; exit 1; fi; go vet ./...'

check: setup ## lint, test, hooks/test-guard.sh y SHA-256 del Excel; distinto de 0 si algo falla
	@echo "##### [1/4] lint #####"
	@$(MAKE) --no-print-directory lint || { echo "CHECK FALLO en el paso 1/4: lint"; exit 1; }
	@echo "##### [2/4] test #####"
	@$(MAKE) --no-print-directory test || { echo "CHECK FALLO en el paso 2/4: test"; exit 1; }
	@echo "##### [3/4] hooks/test-guard.sh #####"
	@bash hooks/test-guard.sh || { echo "CHECK FALLO en el paso 3/4: test-guard"; exit 1; }
	@echo "##### [4/4] SHA-256 del Excel #####"
	@$(MAKE) --no-print-directory _excel || { echo "CHECK FALLO en el paso 4/4: SHA-256 del Excel"; exit 1; }
	@echo "CHECK OK: los cuatro pasos pasaron"

_excel:
	@if command -v sha256sum >/dev/null 2>&1; then sha256sum -c docs/contexto/excel-sha256.txt; else shasum -a 256 -c docs/contexto/excel-sha256.txt; fi

reset-test: setup ## Destruye y recrea SOLO el proyecto catalogo-test
	$(COMPOSE_TEST) down -v
	$(COMPOSE_TEST) up -d --wait db

reset-dev: ## Destruye los datos de desarrollo (exige CONFIRMAR=si)
	@if [ "$(CONFIRMAR)" != "si" ]; then echo "reset-dev borra el volumen de desarrollo. Repita con: make reset-dev CONFIRMAR=si"; exit 1; fi
	$(COMPOSE) down -v

import: setup ## Importa data/CatalogoServicios.xlsx (montado :ro). Repetible sin duplicar
	$(COMPOSE) run --rm --build app importar

seed-demo: setup ## Crea la estructura DEMO y las cuentas de evaluacion (idempotente)
	$(COMPOSE) run --rm --build app sembrar-demo

test-persistence: setup ## P12: reinicia contenedores sin borrar volumen y comprueba que el dato persiste (catalogo-test)
	@bash scripts/persistencia.sh

evidence: setup ## Ejecuta make check y guarda la salida en docs/evidencias/pruebas/<fecha>-<commit>.txt
	@bash scripts/evidencia.sh

e2e: setup ## Prueba de extremo a extremo (Playwright en contenedor, pila catalogo-test). No entra en check
	@# Siembra datos en catalogo_test y, pase o falle, al final destruye SOLO ese proyecto
	@# para dejar la base sin catalogo, que es lo que exigen las pruebas de Go.
	@$(COMPOSE_E2E) up -d --wait db \
	  && $(COMPOSE_E2E) run --rm --build app migrar \
	  && $(COMPOSE_E2E) run --rm app importar \
	  && $(COMPOSE_E2E) run --rm app sembrar-demo \
	  && $(COMPOSE_E2E) up -d --build --wait app \
	  && $(COMPOSE_E2E) run --rm --build e2e; \
	rc=$$?; \
	echo "== e2e: limpiando el proyecto catalogo-test =="; \
	$(COMPOSE_E2E) down -v; \
	exit $$rc
