SHELL := /bin/bash
.DEFAULT_GOAL := check

GO_FILES := $(shell find . -name '*.go' -not -path './web/*' -not -path './.git/*' 2>/dev/null)

.PHONY: check fmt vet test skills data datos-docker servicio bench bench-api evaluacion informe imagen cluster prueba-caida

## check: todo lo que el CI exige antes de abrir un PR
check: skills fmt vet test

fmt:
	@if [ -n "$(strip $(GO_FILES))" ]; then \
		out=$$(gofmt -l $(GO_FILES)); \
		if [ -n "$$out" ]; then echo "gofmt pendiente en:"; echo "$$out"; exit 1; fi; \
		echo "gofmt: ok"; \
	else echo "gofmt: sin archivos Go todavía"; fi

vet:
	@if [ -f go.mod ] && [ -n "$(strip $(GO_FILES))" ]; then go vet ./... && echo "vet: ok"; \
	else echo "vet: sin código Go todavía"; fi

test:
	@if [ -f go.mod ] && [ -n "$(strip $(GO_FILES))" ]; then go test -race -count=1 ./...; \
	else echo "test: sin código Go todavía"; fi

## skills: las skills de Antigravity (.agents) y Claude Code (.claude) deben ser idénticas
skills:
	@bash scripts/check-skills-sync.sh

## data: descarga el dataset de dengue y verifica su sha256
data:
	@bash scripts/descargar-datos.sh

## datos-docker: Parte A en Docker (perfil datos): CSV → silver → gold + shards → modelo.json
datos-docker:
	mkdir -p data reports
	USUARIO="$$(id -u):$$(id -g)" docker compose -f deploy/docker-compose.yml --profile datos up --build entrenar

## servicio: SPA + API + clúster + MongoDB + Redis (requiere deploy/.env y data/gold con modelo.json)
servicio:
	API=:8080 USUARIO="$$(id -u):$$(id -g)" docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.api.yml up --build

bench:
	@go run ./cmd/benchmark

bench-api:
	@go run ./cmd/benchapi

## evaluacion: ROC, calibración y riesgo cruzado sobre test → reports/evaluacion.json (#59)
evaluacion:
	@go run ./cmd/evaluacion

## imagen: construye la imagen del clúster (coordinator y node; la Parte A va en cc65-tf-datos)
imagen:
	docker build -f deploy/Dockerfile -t cc65-tf .

## cluster: entrena en Docker con coordinador + NODOS nodos (requiere data/gold con shards)
cluster:
	USUARIO="$$(id -u):$$(id -g)" docker compose -f deploy/docker-compose.yml up --build --exit-code-from coordinator

## prueba-caida: mata un nodo a mitad del entrenamiento y exige el mismo modelo
prueba-caida:
	@USUARIO="$$(id -u):$$(id -g)" bash scripts/prueba-caida.sh

## informe: secciones con cifras, figuras y el Word (requiere entregables/plantilla.docx)
informe:
	@go run ./cmd/informe
	@go run ./cmd/word
	@echo "Con Word instalado: powershell -ExecutionPolicy Bypass -File scripts/word-final.ps1"

## web: tipos, tests y build de la SPA (lo mismo que el job «web» del CI)
.PHONY: web
web:
	@cd web && npm ci --no-audit --no-fund && npm run typecheck && npm test && npm run build
