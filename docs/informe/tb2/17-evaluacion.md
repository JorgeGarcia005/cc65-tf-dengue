# 17. Evaluación experimental

Esta sección evalúa el sistema en tres dimensiones: qué tan bien predice el modelo (17.1 y 17.2), si cada función hace lo que debe (17.3) y si es rápido y escala (17.4). Todas las cifras se calculan sobre datos que el modelo no vio al entrenar y salen de reportes generados por código.

## 17.1 Datos de prueba y protocolo

- **Conjunto de prueba.** <!--v:miles .Eval.FilasTest-->190 819<!--/v--> casos de 2011 a 2025 (el <!--v:pct .S.M.S.FracTest-->20,00<!--/v--> % del total), separados antes de entrenar por un *hash* de su posición (sección 8.1). Ni las medias y desviaciones de la estandarización ni los pesos del modelo los usan.
- **Modelo evaluado.** El que sirve la API en el momento de la evaluación. `cmd/evaluacion` registra la versión del modelo y la huella SHA-256 de cada archivo que leyó, y el paso de evidencias comprueba que esa versión es la misma que publica la API: no se evalúa un modelo distinto del desplegado.
- **Qué se calcula.** Las métricas de clasificación con el umbral del modelo, la curva ROC y su área por el método del trapecio, la calibración del riesgo por deciles con intervalos de confianza y el cubo de riesgo por departamento, semana y etapa de vida que usan la sección 18 y la SPA.
- **Reproducibilidad.** `go run ./cmd/evaluacion` produce el mismo `reports/evaluacion.json` byte a byte con los mismos datos y el mismo modelo; un test lo comprueba.

## 17.2 Métricas del modelo

<!-- generado:plantilla:metricas -->
Tabla: Métricas del modelo en el conjunto de prueba {#tab:metricas}

| Métrica | Valor | Qué mide |
|---|---:|---|
| AUC-ROC | 0,6221 | Probabilidad de que un caso con signos de alarma reciba más riesgo que uno sin ellos (0,5 = azar; 1 = perfecto) |
| Sensibilidad (*recall*) | 61,91 % | De los casos con signos de alarma, cuántos marca el modelo |
| Precisión | 16,97 % | De los casos que marca, cuántos tienen signos de alarma |
| F1 | 0,266 | Media armónica de precisión y sensibilidad |
| Exactitud | 57,10 % | Casos clasificados correctamente, de ambas clases |
| Brier (riesgo calibrado) | 0,1078 | Error cuadrático medio entre el riesgo y lo ocurrido (menor es mejor) |

Nota: 190 819 casos de prueba, de los cuales 24 002 tienen signos de alarma o son graves; umbral de clasificación 0,5 sobre la probabilidad ponderada. Modelo `655c9dc770fa`. Fuente: `reports/evaluacion.json`, generado por `cmd/evaluacion`.

Tabla: Matriz de confusión en el conjunto de prueba {#tab:confusion}

| | Predicho: con signos de alarma | Predicho: sin signos de alarma |
|---|---:|---:|
| Real: con signos de alarma o grave | 14 859 (verdaderos positivos) | 9 143 (falsos negativos) |
| Real: sin signos de alarma | 72 713 (falsos positivos) | 94 104 (verdaderos negativos) |

Nota: Umbral 0,5. Fuente: `reports/evaluacion.json`.
<!-- /generado -->

**Poder de discriminación.** El AUC de <!--v:cuatro .Eval.Metricas.AUC-->0,6221<!--/v--> indica que el modelo ordena el riesgo mejor que el azar, pero de forma **moderada**. Es lo esperable con las variables disponibles: el conjunto de datos abiertos solo informa edad, sexo, lugar y momento de la notificación. No incluye los factores que la literatura asocia con la gravedad del dengue, como comorbilidades, infección previa por otro serotipo o resultados de laboratorio (Huang et al., 2020). El modelo no sirve para decidir sobre un paciente individual, y no se propone para eso.

**Sensibilidad frente a precisión.** Con el umbral del modelo, se detecta el <!--v:pct .Eval.Metricas.Recall-->61,91<!--/v--> % de los casos con signos de alarma, a costa de una precisión baja: muchos casos marcados no los tienen. Es consecuencia de los pesos por clase (sección 8.2), que priorizan no perder casos graves. Para la pregunta del proyecto, que es de planificación y no de diagnóstico, lo importante no es la clasificación de cada caso sino que el **riesgo agregado** sea correcto, y eso lo mide la calibración.

**Calibración.** La @tab:calibracion compara, por deciles de riesgo, el riesgo que predice el modelo con la proporción de casos que realmente tuvieron signos de alarma. Si en cada decil la tasa observada está cerca del riesgo medio, sumar los riesgos de un grupo de casos da una buena estimación de cuántos tendrán signos de alarma. En los datos de prueba, el riesgo medio va de <!--v:pct (index .Eval.Calibracion 0).Riesgo-->6,33<!--/v--> % en el primer decil a <!--v:pct (index .Eval.Calibracion 9).Riesgo-->21,55<!--/v--> % en el último, y la tasa observada lo sigue de cerca en todos los deciles: el modelo distingue grupos de casos con riesgo varias veces distinto, aunque no pueda separar con certeza un caso de otro. Esa suma, los **casos esperados**, es la cifra con la que la sección 18 responde la pregunta de impacto.

<!-- generado:plantilla:calibracion -->
Tabla: Calibración del riesgo por deciles en el conjunto de prueba {#tab:calibracion}

| Decil | Casos | Riesgo calibrado medio (%) | Tasa observada (%) | IC 95 % de la tasa (%) |
|---:|---:|---:|---:|---|
| 1 | 19 081 | 6,33 | 5,50 | [5,18; 5,83] |
| 2 | 19 082 | 7,62 | 7,94 | [7,57; 8,34] |
| 3 | 19 082 | 8,40 | 8,68 | [8,29; 9,09] |
| 4 | 19 082 | 9,53 | 9,67 | [9,26; 10,10] |
| 5 | 19 082 | 11,28 | 11,15 | [10,71; 11,61] |
| 6 | 19 082 | 12,62 | 12,85 | [12,38; 13,33] |
| 7 | 19 082 | 13,80 | 14,56 | [14,06; 15,07] |
| 8 | 19 082 | 15,65 | 15,67 | [15,16; 16,19] |
| 9 | 19 082 | 18,56 | 18,77 | [18,22; 19,33] |
| 10 | 19 082 | 21,55 | 20,99 | [20,42; 21,57] |

Nota: Los casos de prueba se ordenan por riesgo calibrado y se parten en diez grupos de igual tamaño. Un modelo bien calibrado tiene, en cada decil, una tasa observada cercana al riesgo medio. Fuente: `reports/evaluacion.json` (`calibracion`).
<!-- /generado -->

**Qué aprendió el modelo.** Los pesos más grandes (@tab:coeficientes) son de departamentos: el lugar de notificación es lo que más cambia el riesgo, más que la edad o el sexo. Una razón de odds mayor que 1 indica más riesgo que la categoría de referencia; menor que 1, menos. Es coherente con lo que muestra la sección 18: el riesgo por caso varía más entre regiones que entre grupos de edad.

<!-- generado:plantilla:coeficientes -->
Tabla: Las diez variables con más peso en el modelo {#tab:coeficientes}

| Variable | Peso | Razón de odds |
|---|---:|---:|
| `depto_ayacucho` | -0,909 | 0,40 |
| `depto_cusco` | -0,763 | 0,47 |
| `depto_madre_de_dios` | +0,716 | 2,05 |
| `depto_loreto` | +0,640 | 1,90 |
| `depto_san_martin` | +0,631 | 1,88 |
| `depto_ucayali` | +0,572 | 1,77 |
| `depto_ica` | -0,385 | 0,68 |
| `depto_lima` | -0,338 | 0,71 |
| `depto_ancash` | -0,310 | 0,73 |
| `depto_la_libertad` | -0,301 | 0,74 |

Nota: Razón de odds = e^w; mayor que 1 aumenta el riesgo. Las variables continuas están estandarizadas y las categóricas se comparan con su categoría de referencia. Fuente: `reports/entrenamiento.json`.
<!-- /generado -->

La vista de evaluación de la SPA muestra la curva ROC, la curva de calibración y la matriz de confusión con los mismos datos (@fig:evaluacion).

## 17.3 Pruebas de funcionalidad

Cada parte del sistema tiene pruebas automáticas que se ejecutan en cada Pull Request (el catálogo completo está en el Anexo E):

- **Go:** `go test -race ./...` ejecuta las pruebas unitarias y de integración de todos los paquetes con el detector de carreras: limpieza, cálculo del gradiente, equivalencia bit a bit, protocolo y tolerancia a fallos del clúster, API, autenticación, WebSocket y bases de datos (con MongoDB y Redis reales en el CI).
- **SPA:** Vitest prueba las guardas de ruta, el cliente de la API, el reductor del panel, la reconexión del WebSocket y los colores del mapa.
- **Punta a punta:** Playwright recorre la SPA desplegada con un navegador real, con los flujos de cada rol (sección 16).
- **Verificación formal:** Spin verifica el protocolo y sus mutantes (sección 10).
- **Docker:** el CI levanta el clúster, entrena con verificación bit a bit y ejecuta la prueba de caída de un nodo (sección 9).
- **Contra el sistema desplegado:** `cmd/seguridad` (sección 15.5), `cmd/benchapi` (sección 13), la colección de Postman (sección 12.4) y la cadena de las bases de datos (sección 14.4).

La ejecución completa de las pruebas de Go y de la SPA en la máquina de referencia:

<!-- evidencia: evidencias/pruebas.txt -->

## 17.4 Resumen de las pruebas de rendimiento

La @tab:resumen-rendimiento reúne los resultados de las secciones anteriores. Todas las mediciones se hicieron en la misma máquina, con repeticiones e intervalos de confianza, y cada una comprueba en cada repetición que el resultado es correcto, no solo que es rápido.

Tabla: Resumen de las pruebas de rendimiento {#tab:resumen-rendimiento}

| Qué se mide | Resultado | Correcto en cada repetición | Detalle |
|---|---|---|---|
| Limpieza de <!--v:miles .LoaderRef.Filas-->1 065 274<!--/v--> filas | *Speedup* de <!--v:dos (index .LoaderRef.Mediciones 4).Speedup-->3,25<!--/v--> con <!--v:(index .LoaderRef.Mediciones 4).Workers-->16<!--/v--> workers | Silver igual al secuencial | @tab:bench-loader |
| Gradiente de una época (datos reales) | *Speedup* de <!--v:dos (index .GradGold.Mediciones 5).Speedup-->10,10<!--/v--> con <!--v:(index .GradGold.Mediciones 5).Workers-->16<!--/v--> workers | Igual bit a bit | @tab:bench-gradiente-gold |
| Entrenamiento distribuido | *Speedup* de <!--v:dos (index .Cluster.Mediciones .Ultimo).Speedup-->4,80<!--/v--> con <!--v:(index .Cluster.Mediciones .Ultimo).Nodos-->5<!--/v--> nodos (eficiencia <!--v:dos (index .Cluster.Mediciones .Ultimo).Eficiencia-->0,96<!--/v-->) | Igual bit a bit | @tab:cluster |
| API de predicción con JWT | p95 de <!--v:dos (index .APIJWT.Corridas 0).P95-->3,89<!--/v--> ms (caché fría) y <!--v:dos (index .APIJWT.Corridas 1).P95-->3,09<!--/v--> ms (caliente), límite 100 ms | Sin errores; aciertos de caché esperados | @tab:latencia |
| Observar el entrenamiento en vivo | Diferencia de pocos puntos porcentuales por época | Igual bit a bit | @tab:observador |

Nota: Elaboración propia a partir de las tablas citadas.

**Escalabilidad.** El sistema escala en los dos niveles que pide el curso: dentro de un nodo, con goroutines, el cálculo del gradiente aprovecha todos los núcleos físicos (sección 8.5); entre nodos, el entrenamiento se acelera casi en proporción al número de nodos, porque la comunicación es una fracción pequeña y constante de cada época (sección 11.3). La API cumple el requisito de latencia con un margen amplio, aun con autenticación en cada petición.
