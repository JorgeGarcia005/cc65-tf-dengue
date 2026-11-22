# Respuesta — PR-83 · Ronda 1

- **Implementador:** Claude Code · modelo: Claude Opus 5.5
- **Veredicto respondido:** `ronda-1.md` (sobre `e9495ae`)
- **Rama:** `feature/61-roles-cors`

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO | `8b1af63`: `reports/bench_api_jorge-pc_jwt.json`, medido contra el Compose completo (Mongo + Redis, 3 nodos) · `6a0c187`: `apiReal()` ignora `*_jwt.json` |
| H2 | MAYOR | ACEPTO | `33c8c2d`: `leerJSON` lee el cuerpo acotado antes de decodificar · `413` en `POST /api/v1/entrenamientos` |
| H3 | MAYOR | ACEPTO | `33c8c2d`: `normalizarOrigen` estricto · `Rutas()` con `slices.Clone` |
| H4 | MENOR | ACEPTO | `33c8c2d`: tests de 429/500, `Cache-Control` por ruta, CORS real en `TestEvaluacionConservaVaryDeCORS` y parser del OpenAPI · `21f0801`: `-token` con `-correo` |

## H1 — Reporte del p95 con JWT

`reports/bench_api_jorge-pc_jwt.json` (`"autenticacion": "jwt"`, Windows, 12 CPU, 16 clientes, 2000 peticiones por corrida):

| Corrida | p50 ms | p95 ms | p99 ms | Errores | Caché |
|---|---|---|---|---|---|
| fria | 2.28 | 3.89 | 4.64 | 0 | 0 HIT / 2000 MISS |
| caliente | 1.76 | 3.09 | 3.89 | 0 | 2000 HIT / 0 MISS |

Ambas corridas tienen `cumple_limite: true` con `limite_p95_ms: 100`. El comando fue
`BENCH_CLAVE=… go run ./cmd/benchapi -url http://localhost:18080 -correo analista@vigia.pe -exigir-p95 100ms -maquina jorge-pc`.
Se usó el puerto 18080 porque el 8080 de esa máquina estaba ocupado.

Atendí también el aviso preventivo: `apiReal()` (`cmd/informe/regiones.go`) descarta los `*_jwt.json`.
Así `docs/informe/pc4/10-api.md` sigue citando la medición de la PC4 aunque un hostname nuevo ordene
antes que `deez`. `go test -race ./cmd/informe` está en verde y `go run ./cmd/informe` no cambia ningún archivo.

## H2 — 413 con cualquier cuerpo grande

Causa confirmada: el decodificador en flujo devolvía el `SyntaxError` del primer byte antes de
llegar al límite. Ahora `leerJSON` hace `io.ReadAll(http.MaxBytesReader(…, MaxCuerpo))` y después
decodifica desde memoria. Así responde 413 aunque no haya `Content-Length` (cuerpo *chunked*), caso que
la comprobación de `r.ContentLength` sugerida no cubre. El costo es una copia de ≤ 64 KiB, y una consulta pesa
menos de 1 KB.

Con el código anterior, el test nuevo falla igual que la prueba en vivo del juez:

```text
--- FAIL: TestCabecerasLimiteDeCuerpo413
    seguridad_test.go:331: POST /api/v1/predicciones con 70 000 bytes no JSON: 400 {"error":"JSON mal formado"}
    seguridad_test.go:331: POST /api/v1/entrenamientos con 70 000 bytes no JSON: 400 {"error":"JSON mal formado"}
    seguridad_test.go:331: POST /api/v1/auth/login con 70 000 bytes no JSON: 400 {"error":"JSON mal formado"}
```

Con la corrección, los tres responden 413 y conservan `Access-Control-Allow-Origin`. El test también
comprueba que la autorización va antes que el límite: un POST grande sin token recibe 401.
`TestOpenAPISeguridad` exige ahora que todo POST declare `413`, lo que destapó la ausencia en
`POST /api/v1/entrenamientos`, ya agregada.

## H3 — Orígenes estrictos y roles copiados

`normalizarOrigen` rechaza:

- host vacío (`http://:8081`);
- cualquier `*` en el host (`http://*.example.com`);
- `:` sin puerto (`http://localhost:`);
- puertos fuera de 1–65535 o con cero inicial (`:99999`, `:0`, `:08081`).

IPv6 e IPv4 siguen aceptados (`http://[::1]:5173`, `http://192.168.1.20:8081`). `Rutas()` copia los roles.
Con el código anterior, `TestCORSOrigenesValidos` falla en los seis orígenes nuevos, y
`TestMatrizRolesRutasDevuelveCopias` falla con `el rol "intruso" se filtró de una llamada anterior`.

## H4 — Cobertura y precedencia

- `TestCORSEnRespuestas429Y500`: el sexto login fallido recibe 429 con `Retry-After`, ACAO y
  `Expose-Headers`; un pánico recuperado recibe 500 con ACAO y `nosniff`.
- `TestCabecerasCacheSegunRuta`: `riesgo` y `evaluacion` responden con `private, no-cache` y
  `Vary` ∋ {`Origin`, `Accept-Encoding`}; el login, con `no-store`.
- `TestEvaluacionConservaVaryDeCORS` usa `OrigenesCORS` real en vez del middleware falso.
- `seguridadDelOpenAPI` revisa `sc.Err()` y exige `bearerAuth` bajo `securitySchemes` con
  `type: http` y `scheme: bearer`.
- `cmd/benchapi`: `-token` junto con `-correo` es un error explícito («usa -token o -correo, no
  ambos»), porque los dos pueden venir del entorno sin que se note. Lo cubre `TestMedirConLoginUsaJWT`.

## Verificación

```text
$ make check
gofmt: ok · vet: ok · go test -race ./...: todos los paquetes ok (exit 0)
```
