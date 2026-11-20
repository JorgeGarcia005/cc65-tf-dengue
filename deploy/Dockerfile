# syntax=docker/dockerfile:1
# Imagen única del proyecto: todos los comandos de cmd/ en una imagen mínima.
#
#   docker build -f deploy/Dockerfile -t cc65-tf .
#
# Etapa 1: compila. Etapa 2: solo los binarios sobre distroless (sin shell ni
# gestor de paquetes) y con usuario sin privilegios.

FROM golang:1.24-bookworm AS construccion
WORKDIR /src
# Primero el módulo: mientras no cambien las dependencias, esta capa se reutiliza.
COPY go.mod go.su[m] ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
# El contrato OpenAPI se embebe en el binario (go:embed en docs/api).
COPY docs/api ./docs/api
# CGO_ENABLED=0: binarios estáticos, sin libc (requisito de distroless/static).
# -trimpath: sin rutas de la máquina que compiló. -s -w: sin tabla de símbolos
# ni datos de depuración (binarios más pequeños).
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/...

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=construccion /out/ /usr/local/bin/
USER nonroot:nonroot
# 9000: TCP coordinador–nodos (ADR-0005).
EXPOSE 9000
CMD ["/usr/local/bin/coordinator"]
