# Respuesta — PR #46 · Ronda 1

**Implementador:** Claude Code (Claude Opus 5.5) · **Fecha:** 2026-11-20 · **Base:** `fix/45-imagen-40mb`

| Hallazgo | Severidad | Decisión | Evidencia / commit |
|---|---|---|---|
| H1 | MENOR | ACEPTO | `fedb0b7`: el comentario de `make imagen` (`Makefile:40`) dice ahora «construye la imagen del clúster (solo coordinator y node)», igual que `deploy/README.md:9` |

## Aclaración sobre el tamaño de la imagen

La descripción del PR consignaba 24 956 495 bytes medidos en local; la cifra que vale es la del
CI, **15 438 634 bytes**, porque el límite de `.github/workflows/ci.yml:119` se evalúa allí. La
diferencia es del almacenamiento, no de la imagen:

- El runner usa el almacenamiento clásico de Docker: `.Size` es el tamaño de las capas descomprimidas.
- Docker Desktop local usa el almacén de containerd (`driver-type io.containerd.snapshotter.v1`):
  `.Size` es el uso de disco, que suma además las capas comprimidas (`docker image ls --tree`:
  `DISK USAGE 25MB`, `CONTENT SIZE 6.15MB`).

Comprobación independiente: `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"` de
`./cmd/coordinator` y `./cmd/node` da 10 944 674 + 2 920 610 = 13 865 284 bytes, más la base
distroless (~1,5 MB) ≈ 15,4 MB. La comparación válida es CI contra CI: 40 675 746 bytes en
`develop` (`08f8600`) → 15 438 634 bytes. Corregí la descripción del PR.
