# Entregable 2 — Backend (API REST)

API REST de FinTrack escrita en **Go** con **Gin** y el driver oficial de MongoDB.

- Base de la API: `/api/v1`
- Formato de respuesta uniforme en todos los endpoints
- Autenticación con JWT (access + refresh) — *fase 3*

## Librerías empleadas

| Librería | Para qué |
|---|---|
| [`github.com/gin-gonic/gin`](https://github.com/gin-gonic/gin) | Router y manejo de HTTP |
| [`go.mongodb.org/mongo-driver/v2`](https://github.com/mongodb/mongo-go-driver) | Driver oficial de MongoDB |
| [`github.com/joho/godotenv`](https://github.com/joho/godotenv) | Carga del archivo `.env` en desarrollo |
| [`github.com/stretchr/testify`](https://github.com/stretchr/testify) | Aserciones en las pruebas |
| `log/slog` (biblioteca estándar) | Bitácora estructurada |

Pendientes de las fases siguientes: `golang-jwt/jwt/v5` y `golang.org/x/crypto/bcrypt` (fase 3),
`go-playground/validator/v10` (fase 4), `swaggo/swag` + `swaggo/gin-swagger` (fase 6).

## Requisitos

- Go 1.25 o superior
- MongoDB corriendo (`make up` desde la raíz levanta el de desarrollo)

> El módulo declara `go 1.25.0` porque es lo que exigen Gin 1.12 y `golang.org/x/net`.

## Instalación y ejecución

```bash
# Desde la raiz del repositorio
make up                          # levanta MongoDB

cd backend
cp .env.example .env             # ajusta los valores si hace falta
go mod download                  # descarga las dependencias
go run ./cmd/api                 # arranca la API en http://localhost:8080
```

O con el atajo, desde la raíz: `make dev` (Windows: `.\make.ps1 dev`), que levanta Mongo y
arranca la API en un solo paso.

Al arrancar deberías ver:

```
level=INFO msg="iniciando FinTrack" version=0.1.0 modo=debug
level=INFO msg="conectado a MongoDB" base=fintrack
level=INFO msg="indices verificados" cantidad=6
level=INFO msg="servidor escuchando" direccion=:8080
```

### Variables de entorno

Todas están documentadas en [`.env.example`](.env.example). Si falta alguna obligatoria o trae
un valor inválido, **el servidor no arranca** y reporta de una sola vez todo lo que está mal:

```
configuracion invalida:
  - MONGO_URI es obligatoria
  - JWT_SECRETO_ACCESO debe tener al menos 32 caracteres (tiene 5)
```

| Variable | Por defecto | Obligatoria |
|---|---|---|
| `PUERTO` | `8080` | no |
| `GIN_MODO` | `debug` | no (`debug`, `release` o `test`) |
| `MONGO_URI` | — | **sí** |
| `MONGO_BD` | `fintrack` | no |
| `JWT_SECRETO_ACCESO` | — | **sí** (mínimo 32 caracteres) |
| `JWT_SECRETO_REFRESCO` | — | **sí** (distinto del anterior) |
| `JWT_MINUTOS_ACCESO` | `15` | no |
| `JWT_DIAS_REFRESCO` | `7` | no |
| `CORS_ORIGENES` | `http://localhost:5173` | no (separados por coma) |

En `GIN_MODO=release` el servidor además se niega a arrancar si los secretos siguen teniendo el
valor de ejemplo.

## Estructura

El backend está dividido en capas y el flujo va siempre en una dirección:

```
handlers  ->  servicios  ->  repositorios  ->  MongoDB
 (HTTP)       (reglas)       (consultas)
```

| Paquete | Responsabilidad |
|---|---|
| `cmd/api` | Arranque, cableado de las piezas y apagado ordenado |
| `internal/config` | Carga y validación de las variables de entorno |
| `internal/db` | Conexión a MongoDB, ping y creación de índices |
| `internal/errores` | Errores de dominio tipados y catálogo de códigos |
| `internal/respuestas` | Formato uniforme de respuesta (éxito y error) |
| `internal/middleware` | Id de petición, bitácora, recuperación de panics, CORS |
| `internal/handlers` | Traducción entre HTTP y los servicios |
| `internal/rutas` | Registro de endpoints y middlewares |
| `internal/modelos` | Structs y DTOs *(fase 3)* |
| `internal/repositorios` | Único código que consulta MongoDB *(fase 3)* |
| `internal/servicios` | Reglas de negocio *(fase 3)* |

Ningún archivo Go pasa de ~200 líneas: se prefieren varios archivos chicos.

## Formato de respuesta

Éxito:

```json
{ "datos": { "estado": "ok", "mongo": "ok", "version": "0.1.0" } }
```

Listados paginados (fase 4):

```json
{ "datos": [ ... ], "meta": { "pagina": 1, "limite": 20, "total": 120, "total_paginas": 6 } }
```

Error:

```json
{ "error": { "codigo": "CATEGORIA_NO_ENCONTRADA", "mensaje": "La categoria no existe.", "detalles": [] } }
```

El `codigo` es una cadena estable: el frontend decide qué mostrar según el código, no según el
texto del mensaje. El catálogo completo está en
[`internal/errores/codigos.go`](internal/errores/codigos.go).

## Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/v1/health` | Estado del servicio, ping a MongoDB y versión |

El resto de los endpoints llega en las fases 3 a 6.

```bash
curl http://localhost:8080/api/v1/health
```

```json
{"datos":{"estado":"ok","mongo":"ok","version":"0.1.0","hora":"2026-08-01T05:04:57Z"}}
```

Responde `200` si MongoDB contesta y `503` con `"estado":"degradado"` si no, para que un
orquestador pueda dejar de mandarle tráfico.

## Middlewares propios

Se escribieron a mano en vez de usar librerías de terceros: son pocas líneas, se explican
completas y evitan dependencias.

| Middleware | Qué hace |
|---|---|
| `IDPeticion` | Asigna un id único a cada petición y lo devuelve en `X-Request-ID`. Respeta el que ya venga de un proxy |
| `Bitacora` | Registra método, ruta, estado, latencia, ip e id con `log/slog`. Nivel según el resultado: INFO, WARN (4xx) o ERROR (5xx) |
| `Recuperacion` | Atrapa los panics: el servidor sigue vivo, la traza queda en la bitácora y el cliente recibe un 500 con el formato de siempre |
| `CORS` | Autoriza solo los orígenes de `CORS_ORIGENES` y responde el preflight sin llegar al handler |

En desarrollo la bitácora sale en texto; en `release`, en JSON.

## Apagado ordenado

Al recibir `Ctrl+C` (SIGINT) o `SIGTERM` (lo que manda Docker al detener el contenedor), el
servidor:

1. deja de aceptar conexiones nuevas y espera hasta 10 s a que terminen las peticiones en curso,
2. cierra la conexión a MongoDB,
3. sale.

Ese orden importa: si se cerrara MongoDB primero, las peticiones que aún se están atendiendo
fallarían.

```
{"level":"INFO","msg":"apagando de forma ordenada","espera":"10s"}
{"level":"INFO","msg":"servidor HTTP cerrado"}
{"level":"INFO","msg":"conexion a MongoDB cerrada"}
{"level":"INFO","msg":"hasta luego"}
```

## Índices

`internal/db/indices.go` crea los 6 índices en cada arranque, de forma idempotente. Son los
mismos que crea `database/01_crear_colecciones.js`; se repiten aquí para que un despliegue nuevo
los tenga sin depender de que alguien haya corrido el script.

## Pruebas

```bash
make test                        # desde la raiz (Windows: .\make.ps1 test)
go test ./... -cover             # o directamente, desde backend/
```

Las pruebas de `internal/db` son de integración y necesitan MongoDB. Se **saltan solas** si no
está la variable `MONGO_URI_PRUEBAS`, para que `go test ./...` funcione en cualquier máquina:

```bash
make up
MONGO_URI_PRUEBAS="mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin" \
  go test ./internal/db/... -v
```

Usan una base aparte (`fintrack_pruebas_db`) que se borra al terminar, así que nunca tocan los
datos de desarrollo.

Cobertura actual:

| Paquete | Cobertura |
|---|---|
| `internal/config` | 93.5 % |
| `internal/db` | 90.0 % (con MongoDB) |
| `internal/errores` | 100 % |
| `internal/handlers` | 100 % |
| `internal/middleware` | 98.1 % |
| `internal/respuestas` | 100 % |
| `internal/rutas` | 100 % |
