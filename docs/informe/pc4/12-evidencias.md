# 12. Evidencias de la implementación

Todas las capturas se tomaron con el comando a la vista, sobre el archivo real del MINSA (verificado con su
sha256) y el código de la rama `develop` del repositorio.

## 12.1 Pruebas automatizadas y CI

Ejecución de `go test -race ./...` sobre todo el repositorio, con el detector de condiciones de carrera,
incluidos los tests de equivalencia secuencial = distribuido y los de caída de nodos.

<!-- captura: Pruebas con el detector de carreras en consola -->

El CI de GitHub corre en cada *pull request* el formato, el análisis estático, las pruebas con `-race`, las
pruebas de integración contra MongoDB y Redis, el clúster en Docker con la caída de un nodo, la validación
del OpenAPI y la verificación con Spin.

<!-- captura: Checks del CI en verde en un pull request de GitHub -->

## 12.2 Clúster en Docker con los datos reales

`NODOS=3 VERIFICAR=true docker compose -f deploy/docker-compose.yml up --build --exit-code-from coordinator`
con el gold real: registro de los tres nodos, pérdida por época y verificación contra el secuencial
dentro del contenedor (sin `VERIFICAR=true` el coordinador solo entrena y la comparación se hace después
con `cmd/entrenar -comparar`).

<!-- captura: Clúster de un coordinador y tres nodos en Docker Compose -->

`go run ./cmd/entrenar -comparar reports/entrenamiento_cluster.json`: el entrenamiento del clúster tiene los
mismos hiperparámetros, pesos, sesgo e historia de pérdida que el secuencial, bit a bit.

<!-- captura: Comparación bit a bit del clúster con el entrenamiento secuencial -->

`bash scripts/prueba-caida.sh`: se mata un nodo a mitad del entrenamiento; el coordinador reasigna su shard
en la misma época y los pesos finales siguen idénticos a los secuenciales.

<!-- captura: Caída de un nodo y reasignación de su shard -->

`go run ./cmd/benchmark -cluster -gold data/gold/train.bin`: tiempo por época con 1 a 5 nodos (sección 9).

<!-- captura: Benchmark del clúster por número de nodos en consola -->

## 12.3 API: Swagger y Postman

Swagger UI servido por el coordinador en `/docs`, con las rutas de la sección 10.

<!-- captura: Swagger UI de la API en el navegador -->

Colección de Postman ejecutada contra el clúster en Docker: cada petición con su estado y su tiempo de
respuesta por debajo de 100 ms.

<!-- captura: Ejecución de la colección en Postman con los tiempos de respuesta -->

<!-- captura: Predicción en Postman con la cabecera X-Cache -->

`go run ./cmd/benchapi -exigir-p95 100ms` con 16 clientes concurrentes (sección 10.3).

<!-- captura: Benchmark de latencia de la API en consola -->

## 12.4 MongoDB y Redis

Entrenamiento lanzado por la API y consultado por su ID, y la cadena verificada en las bases con
`bash scripts/evidencia-bd.sh <id>`.

<!-- captura: Entrenamiento distribuido lanzado y consultado por la API -->

<!-- captura: Colecciones e índices de MongoDB y claves de Redis de un entrenamiento -->

## 12.5 Verificación formal

`bash spin/verificar.sh`: el modelo sin errores y cada mutante con su contraejemplo (sección 8.7).

<!-- captura: Verificación del modelo Promela y de los mutantes con Spin -->

## 12.6 Repositorio en GitHub

Historial de *pull requests* de la PC4, cada uno con el veredicto del juez y el CI en verde, y las ramas
con Git Flow.

<!-- captura: Pull requests de la PC4 en GitHub -->

## 12.7 Tablero Kanban

Estado de las tareas de la PC4 en el tablero Kanban de GitHub.

<!-- captura: Tablero Kanban del proyecto en GitHub -->
