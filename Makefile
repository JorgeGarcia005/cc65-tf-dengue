SHELL := /bin/bash
.DEFAULT_GOAL := check

GO_FILES := $(shell find . -name '*.go' -not -path './web/*' -not -path './.git/*' 2>/dev/null)

.PHONY: check fmt vet test skills data bench bench-api

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
