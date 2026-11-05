# 9. Evidencias de la implementación

## 9.1 Carga y limpieza concurrente

Ejecución de `go run ./cmd/loader` sobre el archivo completo: filas leídas, descartadas por regla y
tiempo con un solo worker frente al worker pool.

<!-- captura: Ejecución del loader concurrente en consola -->

## 9.2 Entrenamiento del modelo

Ejecución de `go run ./cmd/entrenar`: métricas del modelo en el conjunto de prueba y verificación de que
el gradiente del worker pool es idéntico al secuencial.

<!-- captura: Entrenamiento de la regresión logística en consola -->

## 9.3 Speedup del cálculo del gradiente

Ejecución de `go run ./cmd/benchmark -gold data/gold/train.bin`: tiempo por época según el número de
workers.

<!-- captura: Benchmark del gradiente en consola -->

## 9.4 Pruebas automatizadas

Ejecución de `go test -race ./...` sobre todo el repositorio, con el detector de condiciones de carrera.

<!-- captura: Pruebas con el detector de carreras en consola -->

## 9.5 Repositorio en GitHub

Historial de commits y ramas del repositorio, organizado con Git Flow.

<!-- captura: Historial de commits y ramas en GitHub -->

## 9.6 Tablero Kanban

Estado de las tareas del proyecto en el tablero Kanban de GitHub.

<!-- captura: Tablero Kanban del proyecto en GitHub -->
