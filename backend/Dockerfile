# Imagen del backend. Dos etapas: una compila y la otra solo ejecuta.
#
# La etapa de compilacion trae el compilador de Go, el codigo fuente y las
# dependencias: unos 800 MB. Nada de eso hace falta para correr un binario ya
# compilado, y todo lo que viaja a un servidor es superficie de ataque. La
# imagen final se queda en unos 25 MB y no contiene ni el codigo fuente ni el
# compilador.

# --- Etapa 1: compilar ------------------------------------------------------
FROM golang:1.25-alpine AS compilacion

WORKDIR /origen

# Las dependencias se copian y se descargan ANTES que el codigo.
#
# Docker guarda en cache cada instruccion y la invalida en cuanto cambia algo de
# lo que copia. Si go.mod y go.sum vinieran junto con el resto, cambiar una sola
# linea de un handler obligaria a volver a descargar todos los modulos.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# VERSION la pasa docker-compose (o el CI) con --build-arg. Sin ella queda el
# valor por defecto que trae config.go.
ARG VERSION=0.1.0

# CGO_ENABLED=0 produce un binario estatico, sin dependencias de la libc del
# sistema. Es lo que permite que la imagen final sea Alpine (que usa musl y no
# glibc) sin que nada se rompa.
#
# -trimpath quita las rutas de la maquina de compilacion del binario.
# -s -w quitan la tabla de simbolos y la informacion de depuracion: unos 25 %
# menos de tamaño a cambio de trazas de panic menos detalladas.
ENV CGO_ENABLED=0 GOOS=linux
RUN go build \
    -trimpath \
    -ldflags="-s -w -X github.com/FNTR3455234/FinTrack/backend/internal/config.Version=${VERSION}" \
    -o /bin/api ./cmd/api

# --- Etapa 2: ejecutar ------------------------------------------------------
FROM alpine:3.21

# ca-certificates para poder hablar por TLS (por ejemplo con un MongoDB Atlas);
# wget lo usa el HEALTHCHECK de aqui abajo y ya viene en busybox.
#
# Se prefiere Alpine a `scratch` justamente por eso: una imagen vacia no tiene
# con que comprobar su propia salud ni con que entrar a mirar cuando algo falla.
# El binario es estatico, asi que de la distribucion no se usa nada mas.
RUN apk add --no-cache ca-certificates tzdata

# Usuario sin privilegios. Por defecto un contenedor corre como root, y si
# alguien consigue ejecutar codigo dentro, lo hace como root.
RUN adduser -D -u 10001 fintrack

COPY --from=compilacion /bin/api /usr/local/bin/api

USER fintrack
EXPOSE 8080

# El propio /health ya comprueba que MongoDB responde, asi que este healthcheck
# distingue "el proceso vive" de "el servicio de verdad funciona". Compose lo usa
# para no arrancar el frontend antes de tiempo.
HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 \
    CMD wget --quiet --spider http://127.0.0.1:8080/api/v1/health || exit 1

# Sin shell de por medio (forma exec): asi el proceso es el PID 1 y recibe
# directamente el SIGTERM que manda Docker al detener el contenedor, que es lo
# que dispara el apagado ordenado de main.go.
ENTRYPOINT ["/usr/local/bin/api"]
