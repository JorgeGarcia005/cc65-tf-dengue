# ADR-0004 — Variables, split y shards del gold

- **Estado:** Aceptado (24/10/2026) · Issue #6

## Decisiones

| Tema | Decisión | Razón |
|---|---|---|
| Años de entrenamiento | Desde 2011 | La categoría *con signos de alarma* (OMS 2009) es marginal en 2010 y no existe antes (§2.3 del informe) |
| Objetivo | `clase ≥ 1` (A97.1 o A97.2) | Indicador de necesidad de hospitalización (ADR-0003) |
| `tipo_dx` | **Excluido** | Los casos graves se confirman en laboratorio con más frecuencia: usarlo filtraría el objetivo |
| Edad | z-score + etapa de vida MINSA one-hot | El z-score da la tendencia; las etapas capturan la forma no lineal (niños con más riesgo) |
| Semana | seno y coseno de 2π(s−1)/52 | La semana 52 queda cerca de la semana 1 (estacionalidad cíclica) |
| Año | z-score | Tendencia; permite predecir años futuros (un one-hot no extrapola) |
| Departamento | one-hot; < 100 casos en train → `OTROS` | Evita columnas casi vacías (Arequipa, Apurímac, Moquegua) |
| Estandarización | Medias y desviaciones **solo del train** | Evita fuga de información del test |
| Split | 80/20 por hash splitmix64 del índice de fila en el silver | Determinista e independiente de workers y nodos; estratificación aproximada verificada (12,55 % vs 12,58 % de positivos) |
| Formato gold | Binario little endian con cabecera (`internal/gold`) | Carga en ~1 s; incluye `bloque_inicial` para el orden global de reducción |
| Shards | Bloques contiguos completos, repartidos lo más parejo posible | Concatenar los parciales en orden de shard = orden global de bloque → gradiente idéntico bit a bit (`TestShardsReproducenElGradienteGlobal`) |
| Calibración | Corrección de prior: logit − ln(PesoPos/PesoNeg) | Con pesos por clase las probabilidades quedan infladas; la corrección (King y Zeng, 2001) devuelve la escala real para reportar riesgos |

## Consecuencias

- `data/gold/esquema.json` y `data/gold/modelo.json` contienen todo lo que la API necesita para
  transformar una consulta y predecir con las mismas reglas que el entrenamiento.
- El orden del silver es cronológico, así que cada shard concentra un rango de años; para
  descenso de gradiente full-batch esto no cambia el resultado (la suma es la misma).
