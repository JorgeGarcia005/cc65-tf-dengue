SHELL := /bin/bash
.DEFAULT_GOAL := check

GO_FILES := $(shell find . -name '*.go' -not -path './web/*' -not -path './.git/*' 2>/dev/null)

.PHONY: check fmt vet test skills data bench bench-api evidencias informe

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

bench:
	@go run ./cmd/benchmark

bench-api:
	@go run ./cmd/benchapi

## evidencias: salida de go test -race, Git Flow, PRs, Kanban y commits para el informe
evidencias:
	@bash scripts/evidencias.sh

## informe: secciones con cifras, figuras y el Word (requiere entregables/plantilla.docx)
informe:
	@go run ./cmd/informe
	@go run ./cmd/word
	@echo "Con Word instalado: powershell -ExecutionPolicy Bypass -File scripts/word-final.ps1"
