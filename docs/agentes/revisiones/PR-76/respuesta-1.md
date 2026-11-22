# PR-76 · Respuesta a la ronda 1

Implementador: Claude Code (Claude Opus 5.5). Rama `feature/58-auth`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO (dependencia) | No es un defecto del código: el CI no corre porque #74 (#56) no estaba fusionado. Orden de merge acordado con Jorge: #74 → #75 → #76. Tras fusionar #74 se actualiza esta rama con `integracion/tb2` («Update branch») para que corran los 6 jobs. |
| H2 | MAYOR | ACEPTO | `3744ac2` — `b64 = base64.RawURLEncoding.Strict()` para las tres partes y rechazo de `\r`/`\n` antes de partir el token. Casos nuevos en `TestJWTRechazos`: «bits sobrantes en la firma» (`mutarUltimo` cambia el bit sobrante del último carácter) y «salto de línea en la firma» / «retorno de carro» → `ErrMalformado`. |
| H3 | MAYOR | ACEPTO | `3744ac2` — `Verificar` exige `partes[0] == cabeceraB64` (ahora es cierto el comentario de `jwt.go`). Si difiere, `errorDeCabecera` decodifica en modo estricto: `alg` explícito distinto de HS256 → `ErrAlgoritmo`; cualquier otra diferencia (`typ` distinto, `ALG`, `crit`, `null`) → `ErrMalformado`. Casos nuevos para los cuatro. |
| H4 | MAYOR | ACEPTO | `aa88478` — `NuevoFirmador` rechaza `duracion < time.Second` (`exp` va en segundos enteros, RFC 7519). `TestJWTSecretoCorto` prueba 0, −1 s y 500 ms, y que un token de 1 s verifica en el instante de emisión. |
| H5 | MAYOR | ACEPTO | `860e58a` — `Comparar` exige sal de 16 bytes y clave de 32 bytes exactos (el costo ya no se multiplica por el largo y no se aceptan claves truncadas), base64 estricto y rechazo de `\r`/`\n`; `pbkdf2.Key` usa siempre `bytesClave`. `TestClaveFormatoPorDefecto` decodifica y comprueba 16/32 bytes; `TestClaveGuardadoInvalido` agrega hashes reales manipulados (truncado a 1 byte, 320 bytes, sal de 4 bytes, salto de línea) con un hash de control que sí verifica. |
| H6 | MENOR | ACEPTO | `6429a6f` — `Requiere` guarda `slices.Clone(roles)`; `TestMiddlewareRolesNoSeAlteran` muta el slice después de armar el middleware. `TestMiddleware` comprueba `sub` y `rol` en el contexto y agrega «token vencido» (401 con `error="invalid_token"`); `TestMiddlewareConcurrente` mezcla 128 peticiones de admin, analista, token inválido y sin token, y verifica código y reclamos en cada una. |

## Verificación

```text
$ gofmt -l internal/auth        (vacío)
$ go vet ./internal/auth        (vacío)
$ go test -race -count=1 -cover ./internal/auth
ok   github.com/JorgeGarcia005/cc65-tf-dengue/internal/auth   8.1s   coverage: 94.1% of statements
```

El resto de la suite no cambia (solo se tocó `internal/auth`). Lista para la ronda 2 cuando el CI corra sobre `integracion/tb2`.
