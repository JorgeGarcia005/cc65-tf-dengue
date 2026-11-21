# Respuesta — PR-35 · Ronda 1

- **Implementador:** Claude Code · modelo: Claude Fable 5.2
- **Veredicto respondido:** [`ronda-1.md`](ronda-1.md) (APROBADO, 1 hallazgo menor)
- **Fecha:** 2026-11-14

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | MENOR | ACEPTO | `11dc87c` — `Mensaje.Validar` rechaza con `ErrInvalido` una `asignacion` con `lambda`, `peso_pos` o `peso_neg` negativos y un `gradiente` con algún parcial de `peso` ≤ 0. Cuatro casos nuevos en `TestMensajeDeTipoDesconocidoOIncompletoSeRechaza` |

## H1 — por qué se corrige aquí y no en un PR posterior

El juez tiene razón en los dos puntos y la corrección es pequeña (13 líneas), así que entra en este
PR en vez de quedar como deuda:

- **Config negativa.** `ml.NuevoPool` normaliza la configuración y entra en pánico con un valor
  negativo (`internal/ml/logistica.go`, `normalizada`). Sin la validación, un coordinador
  defectuoso tumbaría el proceso del nodo. `NaN` e `±Inf` no necesitan chequeo: no existen en
  JSON y el decodificador ya los rechaza (`TestNumeroFueraDeRangoEnElJSONSeRechaza`).
- **Peso ≤ 0.** Todo bloque tiene al menos una fila y los pesos por clase son positivos (un 0 se
  normaliza a 1), así que un parcial legítimo siempre tiene `peso` > 0. Exigirlo por parcial
  garantiza además que el peso total que divide `ml.Finalizar` sea positivo.

El test de ida y vuelta bit a bit usaba `-0` y valores negativos como `peso` de prueba; ahora usa
su valor absoluto más el menor subnormal, para seguir cubriendo subnormales y `MaxFloat64` en ese
campo. Los valores con signo se siguen probando en `grad`, `perdida` y `w`.

## Verificación

```text
skills: ok
gofmt: ok
vet: ok
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/benchmark	1.065s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/entrenar	1.012s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/cmd/word	1.032s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/cluster/protocolo	3.227s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/estadistica	1.021s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/features	1.009s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/gold	1.041s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ingest	4.117s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/maquina	1.015s
ok  	github.com/JorgeGarcia005/cc65-tf-dengue/internal/ml	7.410s
```

## Efecto en el PR #36

Su test `TestNodoFalsoConGradienteDeFormaIncorrectaSeDescarta` envía un parcial con `peso` 0;
tras este cambio ese mensaje se rechaza antes, en el protocolo. Se ajusta al rebasar #36 sobre
`develop` cuando este PR esté mergeado.
