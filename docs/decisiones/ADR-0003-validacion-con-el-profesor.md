# ADR-0003 — Validación del caso con el profesor

- **Estado:** Aceptado (24/10/2026, en clase)
- **Contexto:** issue #4 (perfil del dataset real)

## Contexto

La tabla de casos del enunciado TB2 describe el caso 1 como "Casos positivos acumulados y
fallecidos (MINSA - OGDN)", con un volumen estimado de más de 2,5 M filas y la predicción de
"severidad / hospitalización del paciente según comorbilidades y geolocalización". El perfil
del dataset elegido (`reports/perfil_bronze.json`) muestra diferencias con esa descripción:

| Punto | Enunciado | Dataset de dengue (MINSA/CDC) |
|---|---|---|
| Volumen | > 2,5 M (estimado de la tabla); > 1 M exigido en el Alcance | 1 065 274 filas |
| Severidad / hospitalización | Variable objetivo | No hay hospitalización; sí la clasificación clínica A97.0 / A97.1 / A97.2 |
| Comorbilidades | Variables predictoras | No existen en el dataset abierto |
| Geolocalización | Variables predictoras | Departamento, provincia, distrito y ubigeo |

Además, en el portal solo está disponible "Fallecidos por COVID-19"; "Casos positivos" no
tiene un archivo descargable y ninguno trae comorbilidades.

## Consultas y respuestas del profesor

| # | Pregunta | Respuesta |
|---|---|---|
| 1 | ¿Se acepta dengue con 1,07 M filas, dado que el Alcance exige > 1 M? | Sí |
| 2 | ¿Vale "con signos de alarma o grave" como indicador de severidad / necesidad de hospitalización? | Sí |
| 3 | Sin comorbilidades en los datos abiertos, ¿basta con geolocalización, edad, sexo y estacionalidad? | Sí |
| 4 | ¿Se permiten los drivers de MongoDB y Redis (terceros)? | Sí |

## Decisión

- Se mantiene el caso 1 con el dataset de dengue y regresión logística distribuida.
- La carga y la limpieza procesan las 1 065 274 filas; el entrenamiento usa los años en los
  que existe la categoría "con signos de alarma" (desde 2010, clasificación OMS 2009).
- `AGENTS.md` §3.2 se actualiza: los drivers de MongoDB y Redis quedan permitidos; el resto
  del núcleo sigue en stdlib.

## Consecuencias

- El informe debe explicar por qué el objetivo es un indicador de necesidad de
  hospitalización y declarar la ausencia de comorbilidades como limitación.
- El volumen de entrenamiento (~956 mil filas desde 2011) queda por debajo de 1 M; el
  requisito de > 1 M se cumple en la carga y limpieza concurrentes.
