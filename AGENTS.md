# AGENTS.md — Instrucciones para todos los agentes

Este archivo es la **fuente única de verdad** para cualquier agente que trabaje en el repo
(Claude Code, Antigravity/Gemini, Codex u otro). `CLAUDE.md` y `.agents/rules/` solo agregan
lo específico de cada rol. Si algo aquí contradice a otro archivo, **gana este**.

## 1. Qué es

Trabajo Final (TB2) de **CC65 Programación Concurrente y Distribuida** (UPC, 2026-20).
Sistema distribuido en **Go** que entrena una **Regresión Logística Distribuida** sobre
~1 millón de casos de dengue notificados en el Perú (MINSA / CDC Perú, 2000-2025) para
estimar el riesgo de **dengue con signos de alarma o grave**.

**Pregunta de impacto social (ODS 3):** ¿en qué distritos, semanas epidemiológicas y grupos
de edad conviene reforzar camas, vigilancia y cerco epidemiológico por dengue con signos de
alarma?

**Dataset:** `datos_abiertos_vigilancia_dengue_2000_2025.csv` de
<https://www.datosabiertos.gob.pe/dataset/vigilancia-epidemiol%C3%B3gica-de-dengue>.
Columnas: `departamento, provincia, distrito, localidad, enfermedad, ano, semana, diagnostic,
tipo_dx, diresa, ubigeo, localcod, edad, tipo_edad, sexo`.
**Objetivo binario:** `enfermedad` contiene "CON SIGNOS DE ALARMA" o "GRAVE" → 1; "SIN SIGNOS" → 0.

## 2. Arquitectura (enunciado TB2, Fig. 1)

```
CSV (>1M) ─► A. loader (worker pool) ─► gold + reporte de calidad
                                          │
                                          ▼
SPA (D) ◄─ WebSocket/REST ─► C. coordinator (API HTTP + parameter server)
                               │   ▲  TCP (JSON por línea)
                               ▼   │
                         B. nodos ML × N (shard local, goroutines)
                               │
                MongoDB (persistente) · Redis (caché de predicciones)
```

| Parte | Carpeta | Qué hace |
|---|---|---|
| A | `cmd/loader`, `internal/ingest` | Lee el CSV por bloques de bytes, limpia y valida en paralelo, escribe el gold y el shard de cada nodo |
| B | `cmd/node`, `internal/ml` | Cada nodo calcula el gradiente de su shard con un worker pool y lo envía al coordinador por TCP |
| C | `cmd/coordinator`, `internal/api`, `internal/cluster` | Orquesta épocas, agrega gradientes, actualiza pesos, expone REST + WebSocket, JWT, Mongo, Redis |
| D | `web/` | SPA (React + Vite): login, dashboard, entrenamiento en vivo, mapa de riesgo, panel admin del clúster |
| — | `deploy/` | `docker-compose.yml`: web, coordinator, node ×N, mongo, redis |
| — | `spin/` | (plus) Modelo Promela del protocolo coordinador–nodos |
| — | `docs/informe/` | Fuente del informe (secciones por entregable) |

## 3. Restricciones no negociables

1. **Go para todo el backend** y todo contenerizado; se despliega con `docker compose up`.
2. **Núcleo con biblioteca estándar:** loader, ML, protocolo TCP y API usan solo stdlib.
   Terceros permitidos únicamente para drivers de MongoDB y Redis (pendiente de confirmar
   con el profesor; ver `docs/decisiones/`). Nada de librerías de ML.
3. **Determinismo:** los bloques de datos tienen límites fijos (independientes del número de
   workers o nodos) y las sumas parciales se reducen **en orden de índice de bloque**. El
   gradiente distribuido debe ser **idéntico bit a bit** al secuencial. Hay un test que lo exige.
4. **Sin condiciones de carrera:** todo test de concurrencia corre con `go test -race`.
   Prohibido el estado compartido mutable sin mutex/canal; preferir acumuladores privados.
5. **API de predicción < 100 ms** (p95), medido con un benchmark reproducible, no a ojo.
6. **Ninguna cifra a mano** en el informe: toda tabla o número sale de `reports/*.json`
   generado por código.
7. **Datos no versionados:** `data/` está en `.gitignore`. Se versiona el script que los baja
   y el checksum (`data/MANIFEST.sha256`).

## 4. Git Flow (la rúbrica lo evalúa)

- Ramas: `main` (solo releases), `develop` (integración), `feature/<issue>-<slug>`,
  `fix/<issue>-<slug>`, `release/<entregable>`.
- Nada entra directo a `develop` ni a `main`: **todo por Pull Request**.
- Un PR = un issue = una unidad revisable (idealmente < 400 líneas de código).
- Commits en español, imperativo, convencionales: `feat(ml): agrega reducción ordenada`.
- Release por entregable con tag: `pc3`, `pc4`, `tb2`.
- Cada issue vive en el tablero Kanban del repo (Backlog → En curso → En revisión → Hecho).
- Los tres integrantes deben tener commits propios en cada entregable.

## 5. Roles del ecosistema multiagente

| Rol | Quién | Responsabilidad |
|---|---|---|
| **Orquestador** | Claude (chat del Project "Concurrente TF") | Divide el trabajo en issues, asigna, decide cuando hay desacuerdo, integra |
| **Implementador** | Claude Code | Escribe código + tests + sección del informe; abre el PR; responde al juez |
| **Juez** | Antigravity (Gemini) | Revisa cada PR contra la rúbrica y estas reglas; emite veredicto con evidencia |
| **Equipo humano** | Jorge, José, Marco | Aprueban el merge, entienden y defienden cada parte en la sustentación |

Protocolo completo: [`docs/agentes/PROTOCOLO.md`](docs/agentes/PROTOCOLO.md). Resumen:

1. El orquestador crea el issue con criterios de aceptación verificables.
2. El implementador trabaja en `feature/<n>-…`, corre `make check` y abre el PR hacia `develop`.
3. El juez escribe `docs/agentes/revisiones/PR-<n>/ronda-<k>.md` con la plantilla
   `VEREDICTO.md`: puntaje por criterio, hallazgos con evidencia (archivo:línea, salida de test)
   y decisión `APROBADO` | `CAMBIOS` | `RECHAZADO`.
4. El implementador responde cada hallazgo (`ACEPTO` + commit, o `REBATO` + evidencia).
5. Máximo **3 rondas**. Si no hay acuerdo, decide el orquestador y lo deja escrito.
6. Merge solo con veredicto `APROBADO`, CI verde y aprobación de un humano.

**Reglas del debate:** se discute con evidencia, no con opiniones. Un hallazgo sin
archivo:línea, comando reproducible o cita del enunciado no cuenta. Ningún agente puede
aprobar su propio trabajo.

## 6. Definición de "hecho" por entregable

**PC3 — sáb 07/11/2026 23:30:** caso y pregunta de impacto social; limpieza y EDA con
evidencias; diseño del modelo distribuido (ecuaciones del gradiente y de la agregación);
paralelización en Go (worker pool del loader y del cálculo de gradiente, justificación de
patrones); implementación secuencial vs concurrente con tests `-race` y speedup medido.

**PC4 — sáb 21/11/2026 23:30:** clúster de nodos por TCP en Docker y justificación de la
topología (estrella con parameter server síncrono); API con Swagger/OpenAPI y latencia
< 100 ms demostrada con Postman y benchmark; MongoDB + Redis con diseño de entidades;
informe con PC3 corregido.

**TB2 — dom 29/11/2026 23:30:** SPA integrada con la API (JWT, CORS); evaluación
experimental (métricas del modelo, escalabilidad por número de nodos, respuesta a la
pregunta de impacto social); informe final con PC3 + PC4 corregidos; video ≤ 6 min con los
tres integrantes; sustentación en semana 15.

**Rúbrica TB2 (20):** Planteamiento 3 · Implementación 8 · Interfaz 5 · Informe 2 · Video 2.

## 7. Comandos

```bash
make check        # gofmt + go vet + go test -race ./... (obligatorio antes de abrir PR)
make data         # descarga el CSV y verifica el sha256
make bench        # benchmark secuencial vs concurrente → reports/*.json
docker compose -f deploy/docker-compose.yml up --build
```

## 8. Estilo

- Todo en **español**: código (identificadores en español o inglés técnico, consistente por
  paquete), comentarios, commits, informe y mensajes de error.
- Funciones puras y testeables; la red y el disco en capas delgadas.
- Tests primero (TDD) en la lógica concurrente y numérica.
- Cada decisión de diseño no obvia se registra en `docs/decisiones/ADR-XXXX-*.md`.
- Uso de IA declarado: cada PR indica qué agente lo produjo y qué revisó el humano.
