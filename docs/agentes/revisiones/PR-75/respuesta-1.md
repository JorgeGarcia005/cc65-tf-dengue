# PR-75 · Respuesta a la ronda 1

Implementador: Claude Code (Claude Opus 5.5). Rama `feature/57-adr-seguridad`.

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | BLOQUEANTE | ACEPTO (dependencia) | El CI no corre porque #74 (#56) no estaba fusionado. Tras fusionarlo se actualiza esta rama («Update branch») y corren los jobs. |
| H2 | MAYOR | ACEPTO | `5235430` — fila CORS: `Access-Control-Expose-Headers: X-Cache, Retry-After, Location` y `Vary: Origin` en **toda** respuesta de un origen permitido (incluidos 401/403/429), nombres completos `Access-Control-Allow-*`/`Max-Age`, preflight de origen no permitido → 204 sin `Allow-*`, y el preflight se responde antes del `ServeMux`. Nueva fila «Cabeceras» con el orden de la cadena: recuperar → seguridad → CORS → autenticación por modo → mux → handler. |
| H3 | MAYOR | ACEPTO | `5235430` — fila «Tiempo real»: `http.NewResponseController(w).Hijack()`, `SetDeadline(time.Time{})` para quitar los plazos de `NuevoHTTP`, lectura desde el `bufio.Reader` del hijack, una lectora + una única escritora con plazo de 10 s por escritura. Consecuencias: `Shutdown` no cierra conexiones tomadas, así que `Servidor.Cerrar()` cierra el hub con `1001`; extensiones (`permessage-deflate`) se **declinan** omitiendo `Sec-WebSocket-Extensions` en el `101` (RFC 6455 §9.1), y `1002`/`1003`/`1009` quedan para frames ya conectado. El mismo ajuste de plazos y el cierre `1001` se aplican en el código de #63 (PR #78). |
| H4 | MAYOR | ACEPTO | `5235430` — cada `Ruta` declara modo `publica`, `bearer` o `ticket`; el test tabla recorre cada modo. `GET /api/v1/ws`: (1) `Origin` ausente o fuera de la lista → 403 antes de tocar los tickets; (2) consumo atómico bajo mutex (inexistente, vencido o reusado → 401); (3) rol ≠ admin → 403 como defensa en profundidad (la emisión ya filtra al analista). Matriz actualizada. |
| H5 | MENOR | ACEPTO | `5235430` — «Relacionados» con los issues exactos (#58, #60–#65, #67, #69, #70); `hmac.Equal` y base64url estricto en «Token»; correo normalizado en «Usuarios»; IP desde `RemoteAddr` y ventana deslizante de 1 min sobre intentos fallidos (6.º fallo → 429, coherente con #60) en «Login»; `connect-src` armado desde `API_URL` en «Despliegue del front». Placeholders del PR corregidos a `PR-75`. |

Solo cambia documentación; `make check` no se ve afectado.
