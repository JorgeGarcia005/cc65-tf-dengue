# Guía de José para cerrar el informe TB2 (#70)

Esta guía lleva, paso a paso, desde tu PC (deez, Ryzen 7 5800X) hasta el Word final del informe con todas sus cifras, evidencias y capturas. **Los pasos se hacen en orden y uno por uno.** Después de cada paso corres el control:

```bash
bash scripts/paso70.sh
```

El control revisa todos los pasos desde el 1 y **se detiene en el primero que no está hecho**, con la orden exacta para hacerlo. Un paso solo cuenta si su evidencia terminó bien y es posterior a la del paso anterior: si corres un paso fuera de orden, o repites uno sin repetir los siguientes, el control lo marca y no te deja seguir. **No pases al paso siguiente hasta que el control marque [✓] el actual.**

## Reglas

1. **Las evidencias no se editan a mano.** Las escribe `scripts/evidencias-tb2.sh` con fecha, máquina y commit, y el CI rechaza las que no tienen esa cabecera o no terminaron con `# resultado: OK`. Si un paso falla: lees el error al final de `docs/informe/tb2/evidencias/<paso>.txt`, corriges la causa y vuelves a correr **el mismo paso**.
2. **Nada de push ni merge.** Al final le envías a Jorge un *bundle* (paso 18). Tampoco cierras issues ni tocas `develop`, `main` o los tags.
3. **Los commits van a tu nombre** (cuenta `JJva-06`): la rúbrica pide commits de los dos.
4. **No versiones** `deploy/.env`, `data/`, `entregables/` ni claves. El `.gitignore` ya los excluye: no uses `git add -f`.
5. **Si algo no se puede resolver en 15 minutos, detente y escríbele a Jorge** con la salida de `bash scripts/paso70.sh` y el archivo de evidencia. No improvises cambios en el código.
6. **Los benchmarks de limpieza, gradiente y clúster no se repiten.** Ya son de deez con el gold real (`reports/bench_*_deez*.json`) y no dependen de la versión del modelo. Lo que cambió desde la PC4 (la versión del modelo, la medición con JWT, las vistas de la SPA) es lo que regeneran los pasos de esta guía.

## Dónde trabajar

Todo se corre en **WSL (Debian)**, con el repositorio dentro de WSL (no en `/mnt/c/…`, que es lento y cambia las fechas de los archivos) y Docker Desktop con la integración de WSL activada (*Settings → Resources → WSL integration → Debian*). En WSL necesitas:

```bash
sudo apt-get update && sudo apt-get install -y git curl gcc spin make
go version      # 1.24 o más
node -v         # v22
docker compose version
```

## Paso 1. Preparar la rama y los archivos locales

```bash
cd ~
git clone https://github.com/JorgeGarcia005/cc65-tf-dengue.git   # si ya lo tienes en WSL: cd cc65-tf-dengue && git fetch origin
cd cc65-tf-dengue
git config user.name  "José Javier Villanueva Aramburú"
git config user.email "<el correo de tu cuenta JJva-06>"
git switch -c feature/70-evidencias origin/integracion/tb2
mkdir -p data/bronze reports entregables
cp /mnt/c/<ruta>/vigilancia_dengue_2000_2025.csv data/bronze/       # el CSV que ya usaste en la PC3
cp deploy/.env.example deploy/.env
```

En `deploy/.env` completa `JWT_SECRETO` (genéralo con `openssl rand -base64 48`), `VIGIA_ADMIN_CLAVE` y `VIGIA_ANALISTA_CLAVE` (12 caracteres o más cada una). Copia en `entregables/` la plantilla `plantilla.docx` que te pase Jorge y crea `entregables/codigos.txt` con tu código de alumno y el de Jorge, uno por línea.

Si el puerto 8080 u 8081 está ocupado en tu PC, usa otros en **todos** los pasos: `export PUERTO_API=18080 PUERTO_WEB=18081`.

Control: `bash scripts/paso70.sh` → [✓] Paso 1.

## Pasos 2 a 12. Evidencias por script

Cada paso es una sola orden. Córrelas en este orden, con el control después de cada una.

| Paso | Orden | Qué hace | Tiempo aprox. |
|---|---|---|---|
| 2 | `bash scripts/evidencias-tb2.sh entorno` | Versiones de las herramientas, CPU y SHA-256 del CSV | 1 min |
| 3 | `bash scripts/evidencias-tb2.sh parte-a` | Parte A en Docker: `loader` → `gold` → `entrenar`; deja `data/gold/` y el modelo inicial | 5 min |
| 4 | `bash scripts/evidencias-tb2.sh cluster` | Entrenamiento único en el clúster con `VERIFICAR=true`: pesos idénticos bit a bit al secuencial | 3 min |
| 5 | `bash scripts/evidencias-tb2.sh caida` | Mata un nodo a mitad del entrenamiento y exige el mismo modelo | 3 min |
| 6 | `bash scripts/evidencias-tb2.sh servicio` | Levanta el sistema completo con `DEMO=true` y espera a que se registren los nodos. **Desde aquí el sistema queda arriba hasta el paso 15** | 3 min |
| 7 | `bash scripts/evidencias-tb2.sh entrenamiento` | Entrena por la API con los parámetros del modelo inicial y anota la versión publicada | 2 min |
| 8 | `bash scripts/evidencias-tb2.sh benchapi` | Latencia de la API con JWT → `reports/bench_api_deez_jwt.json`; falla si el p95 no es menor que 100 ms | 1 min |
| 9 | `bash scripts/evidencias-tb2.sh bd` | Cadena del entrenamiento en MongoDB y Redis (modelo, entrenamiento, predicciones, caché, usuarios) | 1 min |
| 10 | `bash scripts/evidencias-tb2.sh evaluacion` | `reports/evaluacion.json` con la latencia con JWT; falla si la versión evaluada no es la que sirve la API | 2 min |
| 11 | `bash scripts/evidencias-tb2.sh spin` | Verifica el protocolo con Spin → `reports/spin.json` | 5 min |
| 12 | `bash scripts/evidencias-tb2.sh pruebas` | `go test -race ./...` y los tests de la SPA | 5 min |

Errores comunes:

- **Paso 3 falla en `loader`**: casi siempre es el CSV (nombre distinto o archivo incompleto). El paso 2 ya verificó el SHA-256; si cambiaste el archivo después, vuelve a correr el paso 2 y luego el 3.
- **Paso 6 no ve «la API acepta entrenamientos»**: revisa `docs/informe/tb2/evidencias/servicio.log`. Si dice que falta `JWT_SECRETO` o una clave, corrige `deploy/.env` y repite el paso 6.
- **Paso 10 dice que la versión no coincide**: el modelo de la API no es el de `data/gold/modelo.json`. Repite desde el paso 6 (el control te lo pedirá).
- **Paso 11 o 12 no encuentra `spin` o `gcc`**: instálalos en WSL (sección «Dónde trabajar») y repite solo ese paso; el control te dirá si hace falta repetir los siguientes.

## Paso 13. Seguridad

Si creaste tu rama antes del PR de seguridad y Playwright (el control te lo dirá en el paso 1), primero `git merge origin/integracion/tb2` (sin rebase). Con el sistema arriba:

```bash
bash scripts/evidencias-tb2.sh seguridad
```

`cmd/seguridad` prueba 24 casos desde fuera: login, token ausente, mal formado, con la firma alterada, vencido o de otro secreto, roles, CORS permitido y denegado, cabeceras de seguridad, CSP de la SPA, WebSocket sin ticket, con ticket reutilizado y de un origen ajeno, cuerpo grande (413) y límite de intentos (429). Al final **espera un minuto** a que el límite se libere y comprueba que el login vuelve a funcionar, para no bloquear el paso 14. Escribe `reports/seguridad.json` con lo esperado y lo obtenido. Si algún caso no da lo esperado, **no lo arregles**: es un hallazgo; avísale a Jorge.

## Paso 14. Pruebas de punta a punta y capturas automáticas (Playwright)

Con el sistema arriba y conexión a internet (la primera vez descarga el Chromium de Playwright, y Swagger UI se carga desde `cdn.jsdelivr.net`):

```bash
bash scripts/evidencias-tb2.sh e2e
```

Recorre la SPA como analista y como administrador (9 pruebas) y guarda estas capturas en `docs/informe/tb2/capturas/`. **No las tomes a mano ni les cambies el nombre**: el informe las busca por nombre. La prueba de Inicio falla a propósito si la evaluación no es del modelo que sirve la API: en ese caso repite desde el paso 6.

| Archivo | Qué muestra |
|---|---|
| `e2e-01-login.png` | Inicio de sesión con el error genérico tras una clave errada |
| `e2e-02-analista-403.png` | El analista intenta entrar al panel de administración |
| `e2e-03-inicio.png` | Inicio con las cifras del modelo |
| `e2e-04-mapa.png` | Mapa de riesgo en la semana pico |
| `e2e-05-mapa-departamento.png` | Mapa con el panel de un departamento |
| `e2e-06-prediccion.png` | Predicción de un caso con sus contribuciones |
| `e2e-07-evaluacion.png` | Evaluación: ROC, calibración y matriz |
| `e2e-08-admin-entrenando.png` | Panel de administración durante un entrenamiento (época, pérdida, nodos sanos, p95 de la API) |
| `e2e-09-admin-nodo-caido.png` | Bitácora tras tumbar un nodo: la caída y el reingreso (o la reasignación de su shard) |
| `e2e-10-admin-metricas.png` | Topología con el uso de CPU y la latencia de cada nodo al terminar |
| `e2e-11-swagger.png` | Swagger con el candado de autorización |
| `e2e-12-responsive-375.png` | La SPA a 375 px de ancho |

## Paso 15. Capturas manuales

Solo dos, las que no se pueden automatizar. Con el sistema todavía arriba, en Postman (Windows), importa `docs/api/cc65-tf.postman_collection.json`, cambia la variable `base_url` si usaste otro puerto y pon en `clave` la de `VIGIA_ADMIN_CLAVE`.

| Archivo en `docs/informe/tb2/capturas/` | Qué debe verse |
|---|---|
| `m-postman-runner.png` | *Collection Runner* con la colección completa: todas las pruebas en verde y la columna de tiempo de cada petición |
| `m-postman-prediccion.png` | `POST /api/v1/predicciones` con la pestaña *Authorization* (Bearer), el cuerpo, la respuesta `200` y el tiempo en ms |

Guárdalas en PNG, a pantalla completa de la ventana de Postman, sin recortar la barra de estado (ahí sale el tiempo). Si cuando llegues a este paso el informe pide alguna captura más, el control la listará con su nombre exacto.

Cuando el control marque [✓] el paso 15, ya puedes apagar el sistema: `bash scripts/evidencias-tb2.sh apagar`.

## Paso 16. Informe final

```bash
QUITAR="<el texto que te indique Jorge>" bash scripts/evidencias-tb2.sh informe
```

Regenera las cifras del informe desde los `reports/*.json` nuevos, comprueba que el informe coincide con el repositorio y arma el Word con `cmd/word -final`, que **falla si falta cualquier captura o evidencia**. `QUITAR` borra de la carátula de la plantilla el párrafo con el nombre del integrante que se retiró. El Word queda en `entregables/1ACC0065_TB2_202610_Informe_<código>.docx`, uno por cada código de `entregables/codigos.txt`.

Ábrelo en Word, acepta actualizar los campos (índices y números de página) y revisa que cada captura se vea completa. Si una no se lee, vuelve a tomarla y **repite el paso 16**.

## Paso 17. Commits

```bash
git status
git add docs/informe reports
git commit -m "docs(informe): evidencias, capturas y cifras de la TB2 en deez (#70)"
```

`git status` no debe mostrar `deploy/.env`, `data/` ni `entregables/`. Si aparecen, **no los agregues**: avísale a Jorge.

## Paso 18. Entrega a Jorge

```bash
git bundle create jose-70.bundle origin/integracion/tb2..feature/70-evidencias
bash scripts/paso70.sh      # debe terminar con «Todo completo»
```

Envíale a Jorge `jose-70.bundle`, el `.docx` de `entregables/` y la salida del control. Él revisa, sube la rama y abre el PR; el juez revisa las evidencias como cualquier otro PR.
