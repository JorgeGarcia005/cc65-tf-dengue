# Respuesta — PR-85 · Ronda 1

- **Implementador:** Claude Code · modelo: Claude Opus 5.5
- **Veredicto respondido:** `ronda-1.md`
- **Rama:** `feature/67-inicio-prediccion`

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | MAYOR | ACEPTO | `d1b1960` — `esPrediccion`, `esModelo` y `esEvaluacion` antes de pintar · `proporcion()` sin NaN |
| H2 | MAYOR | ACEPTO | `0849eef` — tablas en `.tabla-envoltura`, `min-width: 0` en el grid, `minmax(min(420px, 100%), 1fr)` y campos del formulario al 100 % |
| H3 | MENOR | ACEPTO | `d8a468d` (Go: `-0` y test sin FMA) · `d1b1960` (`radiogroup`, texto del conteo, sin código muerto) · `0849eef` (borde del fieldset inválido) |

## H1 — Respuestas 200 con otra forma

Validadores en `prediccion/datos.ts` y `inicio/datos.ts`:

- **`esPrediccion`** exige `riesgo`, `probabilidad_ponderada`, `umbral`, `logit` y `sesgo` finitos, `clase` 0/1, los textos y `contribuciones` como arreglo de `{variable, valor, peso, aporte}` numéricos.
- **`esModelo`** exige `version` no vacía y `ano_min` entero.
- **`esEvaluacion`** suma a `version_esquema` lo que la API no valida: métricas, `filas_test`, `positivos_test`, `auc_trapecio`, `brier_calibrado` y el cubo.

Si no cumplen, la vista muestra el `role="alert"` y conserva el formulario o el encabezado. `proporcion(a, b)` devuelve 0 si `b ≤ 0` y la usan la barra de semanas pico y la prevalencia.

Tests nuevos (Vitest, 72 en total):

- **Predicción:** HTML de proxy, JSON sin `contribuciones` y `logit` no numérico dan un aviso y no aparece ningún `[data-resultado]`; un modelo `200 {}` da un aviso sin «undefined»; `esPrediccion` rechaza contribuciones incompletas y `clase: 2`.
- **Inicio:** HTML de proxy, sin `metricas_test`, `prioridades: null` y `version_esquema: 2` dan un aviso y el título se mantiene; `proporcion(5, 0) === 0`.

## H2 — Desborde a 768 y 375 px

Medido con Chrome headless sobre el build, con el `evaluacion.json` real (`scrollWidth` vs `clientWidth` del documento y tarjetas cuyo contenido se sale):

```text
antes   1280px  inicio 1280/1280 (0 tarjetas)  | prediccion 1280/1280
antes    768px  inicio  796/768  (1 tarjeta)   | prediccion  768/768
antes    375px  inicio  786/375  (1 tarjeta)   | prediccion  394/375
después 1280px  inicio 1280/1280 (0)           | prediccion 1280/1280
después  768px  inicio  768/768  (0)           | prediccion  768/768
después  375px  inicio  375/375  (0)           | prediccion  375/375
```

La medición destapó un desborde que el juez no había reportado: el formulario de predicción a 375 px (los `input` tienen ancho intrínseco). Lo corregí con `width: 100%; min-width: 0` en sus campos. A 768 px las dos columnas de Inicio pasan a una (umbral de 420 px), así la tabla de grupos de edad no necesita desplazarse. A 375 px, la de prioridades se desplaza dentro de su tarjeta.

## H3 — Detalles

1. **`"aporte":-0`:** `if aporte == 0 { aporte = 0 }` antes de sumar. `z` no cambia, porque sumar `-0` o `+0` a un `z` no nulo da lo mismo. El test exige ahora `aporte == float64(peso·valor)` y que no haya ceros con signo.
2. **Código muerto:** salió `sigmoide` y también el parámetro `departamentos` de `etiquetaVariable`, que no tenía efecto.
3. **Fieldset del sexo:** ahora tiene `role="radiogroup"` con `aria-labelledby` a su leyenda, y `fieldset[aria-invalid='true']` lleva un borde de error como los inputs. El test busca el `radiogroup` «Sexo» con `aria-invalid`.
4. **Conteo:** el texto pasó a «N variables con aporte 0 no se muestran».
5. **`TestPrediccionCoincideConElModeloEntrenado`:** suma con `float64(m.Pesos[j] * x[j])`, como el modelo, así no depende de que el compilador emita FMA en arm64.
6. **Latencia:** queda anotado para la corrida final de evidencias (#70/#71): regenerar `bench_api_jorge-pc_jwt.json` con el payload actual. La medición en vivo del juez ya dio p95 de 3,93 / 3,57 ms.

## Verificación

```text
$ make check                      → gofmt ok · vet ok · go test -race ./... ok
$ npx tsc -b && npx vitest run    → 72 passed
$ npm run build                   → ok; 0 url(data:) en el CSS
$ go run ./cmd/informe            → regenera el fragmento de 10-api.md (cambió el bucle de la predicción)
```
