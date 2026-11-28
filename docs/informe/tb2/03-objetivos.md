# 3. Objetivos

## 3.1 Objetivo general

Construir en Go un sistema concurrente y distribuido que entrene una regresión logística sobre los casos de dengue notificados por el MINSA, la exponga como un servicio que responda en menos de 100 ms y la presente en una aplicación web, para estimar el riesgo de dengue con signos de alarma o grave e indicar en qué departamentos, semanas epidemiológicas y grupos de edad conviene reforzar camas, vigilancia y cerco epidemiológico.

## 3.2 Objetivos específicos y su verificación

1. **Parte A.** Cargar y limpiar de forma concurrente el archivo completo de casos de dengue, con el mismo resultado que la limpieza secuencial.
2. **Parte B.** Entrenar la regresión logística con el cálculo del gradiente repartido entre goroutines, sin condiciones de carrera y con el mismo gradiente que el cálculo secuencial.
3. **Clúster.** Distribuir el entrenamiento en nodos que se comunican por TCP con un coordinador, desplegados con Docker Compose, tolerante a la caída de un nodo y con el mismo modelo que el entrenamiento secuencial.
4. **Verificación formal.** Verificar con Spin que el protocolo entre el coordinador y los nodos no tiene interbloqueos ni pierde parciales, y que el modelo distingue cada decisión del diseño del error que corrige.
5. **Parte C.** Exponer el modelo en una API REST y WebSocket documentada con OpenAPI, protegida con JWT, que responda en menos de 100 ms, y guardar modelos, entrenamientos y predicciones en MongoDB con Redis como caché.
6. **Parte D.** Presentar los resultados en una SPA que permita iniciar sesión, consultar el riesgo, configurar y lanzar un entrenamiento y seguirlo en tiempo real.
7. **Evaluación e impacto.** Evaluar el modelo sobre datos que no vio al entrenar y responder con él la pregunta de impacto social.

Cada objetivo tiene un criterio que se puede comprobar con un comando o un archivo del repositorio (@tab:objetivos). Los resultados de la tabla se generan desde los reportes JSON al construir el informe.

Tabla: Criterio de verificación y resultado de cada objetivo específico {#tab:objetivos}

| Objetivo | Criterio de éxito | Cómo se verifica | Resultado | Sección |
|---|---|---|---|---|
| 1. Parte A | Más de un millón de filas; silver concurrente igual al secuencial | `go run ./cmd/loader` → `reports/limpieza.json` | <!--v:miles .S.P.Perfil.Filas-->1 065 274<!--/v--> filas; iguales: <!--v:sino .S.L.Equivale-->sí<!--/v-->; speedup hasta <!--v:dos .S.LoaderMax-->3,54<!--/v-->× | 6, 7 |
| 2. Parte B | Gradiente con *worker pool* idéntico bit a bit al secuencial; `go test -race` sin carreras | `go test -race ./internal/ml` y `cmd/benchmark` | Idéntico: <!--v:sino .S.TodoBitABit-->sí<!--/v-->; speedup hasta <!--v:dos .S.GradMax-->10,96<!--/v-->× con <!--v:.S.GradWorkers-->32<!--/v--> workers | 8 |
| 3. Clúster | Modelo distribuido igual al secuencial con 1 a 5 nodos y tras la caída de uno | `VERIFICAR=true make cluster`, `make prueba-caida` y job Docker del CI | Speedup de <!--v:dos (index .Cluster.Mediciones .Ultimo).Speedup-->4,80<!--/v-->× con <!--v:(index .Cluster.Mediciones .Ultimo).Nodos-->5<!--/v--> nodos, igual bit a bit en cada medición | 9, 11 |
| 4. Verificación | Sin errores en el modelo correcto; cada mutante da un contraejemplo | `bash spin/verificar.sh` → `reports/spin.json` | <!--v:miles .SpinEstados-->17 949 492<!--/v--> estados sin errores; <!--v:.SpinMutantes-->4<!--/v--> mutantes con contraejemplo | 10 |
| 5. Parte C | Percentil 95 menor que 100 ms con carga concurrente y JWT | `go run ./cmd/benchapi` y colección Postman | p95 de <!--v:dos (index .APIJWT.Corridas 0).P95-->3,89<!--/v--> ms (caché fría) y <!--v:dos (index .APIJWT.Corridas 1).P95-->3,09<!--/v--> ms (caliente), <!--v:(index .APIJWT.Corridas 0).Errores-->0<!--/v--> errores | 12–15 |
| 6. Parte D | Inicio de sesión, configuración del entrenamiento, resultados en tiempo real | Tests de la SPA (Vitest) y recorrido de punta a punta | Sección 16 y catálogo de pruebas (anexo E) | 16, 17.3 |
| 7. Evaluación | Mejor que el azar (AUC > 0,5) y probabilidad calibrada sobre el conjunto de prueba | `go run ./cmd/evaluacion` → `reports/evaluacion.json` | AUC de <!--v:cuatro .R.evaluacion.metricas_test.AUC-->0,6221<!--/v-->; Brier de <!--v:cuatro .R.evaluacion.brier_calibrado-->0,1078<!--/v--> | 17, 18 |

Nota: El clúster se midió en un <!--v:.Cluster.Maquina.CPU-->AMD Ryzen 7 5800X 8-Core Processor<!--/v--> (<!--v:.Cluster.Maquina.NumCPU-->16<!--/v--> hilos); la API, en la máquina indicada en la sección 13. Los speedup de la Parte A y de la Parte B son el mejor valor medido entre las máquinas de la sección 7.6 y 8.5. Fuente: `reports/*.json`.
