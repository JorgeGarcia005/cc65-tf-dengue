# ADR-0001 — Caso, dataset y algoritmo

- **Estado:** Aceptado (24/10/2026)
- **Decisores:** equipo + orquestador

## Contexto

El enunciado TB2 ofrece 10 casos con datasets del Portal Nacional de Datos Abiertos y exige
más de 1 000 000 de registros procesados concurrentemente, un modelo de ML entrenado de
forma distribuida en un clúster de nodos Go por TCP, API con latencia < 100 ms, MongoDB,
Redis y una SPA. Se verificó en el portal (24/10/2026) qué datasets existen realmente:

| Caso | Dataset en el portal | Volumen | ¿Cumple >1M por registro? |
|---|---|---|---|
| 1 Dengue/COVID | Vigilancia epidemiológica de dengue 2000-2025 (MINSA/CDC) | ~1,0 M filas, 118 MB | Sí |
| 2 Delitos | Denuncias SIDPOL 2018-2026 (conteos agregados) | ~360 k filas | No |
| 3 Accidentes | SUTRAN, accidentes en carreteras 2020-2021 | muy inferior | No |
| 5 Energía | Consumo por suministro, Distriluz | millones | Sí (red neuronal) |
| 10 Aire | SENAMHI, contaminantes horarios Lima | sin confirmar | — |

## Decisión

**Caso 1 con el dataset de dengue y Regresión Logística Distribuida** mediante descenso de
gradiente síncrono con *parameter server* (el coordinador) y *data parallelism* (cada nodo
tiene un shard fijo).

- **Objetivo:** `y = 1` si `enfermedad` es "DENGUE CON SIGNOS DE ALARMA" o "DENGUE GRAVE"
  (CIE-10 A97.1 / A97.2); `y = 0` si es "DENGUE SIN SIGNOS DE ALARMA" (A97.0).
- **Features candidatas:** edad normalizada y por tramos, sexo, departamento (one-hot),
  semana epidemiológica (codificación seno/coseno), año, tipo de diagnóstico.
- **Gradiente:** `∇L(w) = (1/N) Σᵢ (σ(xᵢᵀw) − yᵢ) xᵢ (+ λw)`. Como es una suma, cada nodo
  calcula su suma parcial y el coordinador las agrega: `∇L = (1/N) Σ_nodos Σ_bloques g_b`.
- **Determinismo:** bloques de tamaño fijo con índice global; cada nodo reduce sus bloques
  en orden de índice y el coordinador reduce los nodos en orden de id. Así el resultado es
  idéntico bit a bit al secuencial para cualquier número de workers y nodos.

## Consecuencias

- **A favor:** matemática simple de explicar en la sustentación; la equivalencia
  secuencial = distribuido es demostrable con un test; el protocolo (difundir pesos →
  recoger gradientes → actualizar) es una barrera por época, fácil de modelar en Promela.
- **En contra:** el dataset tiene pocas variables clínicas, así que el poder predictivo será
  moderado; hay desbalance de clases (se usarán pesos por clase y se reportarán AUC y
  recall, no solo exactitud). Se declarará como limitación.
- **Resuelto en ADR-0003:** el profesor aprobó el dataset, el objetivo, las variables y
  los drivers de MongoDB y Redis.
