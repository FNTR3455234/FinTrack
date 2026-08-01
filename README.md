# FinTrack

Aplicación web de finanzas personales y presupuestos: registra tus ingresos y gastos, organízalos
por cuentas y categorías, ponles un límite mensual y mira a dónde se te va el dinero.

> **Estado:** en desarrollo. Fases 0 a 2 de 10 completadas — ver la [tabla de avance](#avance-por-fases).

## Stack

| Capa | Tecnología |
|---|---|
| Base de datos | MongoDB 7 |
| Backend | Go 1.25+ · Gin · driver oficial `mongo-driver/v2` · JWT · bcrypt |
| Frontend | React 18 · Vite · React Router · axios · Recharts · CSS Modules |
| Infraestructura | Docker · Docker Compose |
| Pruebas | testify (Go) · colección de Postman |

## Arranque rápido (desarrollo)

Requisitos: [Docker Desktop](https://www.docker.com/products/docker-desktop/), Go 1.22+ y Node 18+.

```bash
git clone https://github.com/FNTR3455234/FinTrack.git
cd FinTrack

# 1. Levanta MongoDB (la primera vez crea el esquema y carga los datos de ejemplo solo)
make up                 # Windows sin make:  .\make.ps1 up

# 2. Configura el backend (a partir de la fase 2)
cp backend/.env.example backend/.env

# 3. Para recargar el esquema y los datos de ejemplo cuando quieras
make seed
```

Mongo queda escuchando en `localhost:27017` con el usuario `fintrack_admin`. Para detenerlo:
`make down` (los datos se conservan) o `docker compose -f docker-compose.dev.yml down -v` (los borra).

**Usuario de ejemplo:** `demo@fintrack.mx` / `Demo1234!` — trae 6 meses de historia
(3 cuentas, 10 categorías, 120 transacciones y 6 presupuestos).

El stack completo en contenedores (`docker compose up` y listo) llega en la fase 8.

## Atajos

Los mismos targets existen en `Makefile` (Linux/macOS/CI) y en `make.ps1` (Windows sin `make`):

| Atajo | Qué hace | Disponible desde |
|---|---|---|
| `make up` / `make down` | Levanta o detiene MongoDB | fase 0 |
| `make seed` | Recrea el esquema y carga los datos semilla | fase 1 |
| `make dev` | Levanta Mongo y arranca la API | fase 2 |
| `make lint` | `go vet` + `golangci-lint` | fase 2 |
| `make test` | Pruebas de Go con cobertura | fase 3 |
| `make build` | Compila backend y frontend | fase 7 |

`make help` (o `.\make.ps1`) lista todo.

## Documentación

| Documento | Contenido |
|---|---|
| [`docs/arquitectura.md`](docs/arquitectura.md) | Capas del backend, aislamiento por usuario, flujo de autenticación |
| [`docs/decisiones.md`](docs/decisiones.md) | Bitácora de decisiones técnicas y su porqué |
| [`database/modelo.md`](database/modelo.md) | Diagrama entidad-relación, relaciones e índices |

### Entregables

| # | Entregable | README |
|---|---|---|
| 1 | Base de datos: modelo, scripts de creación, semilla y respaldo | [`database/README.md`](database/README.md) ✅ |
| 2 | Backend: API REST en Go | [`backend/README.md`](backend/README.md) ✅ |
| 3 | Frontend: cliente en React | `frontend/README.md` _(fase 7)_ |
| 4 | Pruebas: colección de Postman | `postman/README.md` _(fase 6)_ |

## Avance por fases

| Fase | Contenido | Estado |
|---|---|---|
| 0 | Estructura del repo, `.gitignore`, Compose de desarrollo, Makefile | ✅ |
| 1 | Base de datos: modelo, `$jsonSchema`, índices, semilla, respaldo | ✅ |
| 2 | Backend base: config, conexión, `/health`, errores, middlewares, apagado ordenado | ✅ |
| 3 | Autenticación: registro, login, refresh, perfil, rate limit | ⏳ |
| 4 | CRUD de cuentas, categorías y transacciones con filtros y paginación | ⏳ |
| 5 | Presupuestos, consultas relacionales, resumen, tendencia y alertas | ⏳ |
| 6 | Exportar/importar CSV, Swagger, colección de Postman | ⏳ |
| 7 | Frontend completo | ⏳ |
| 8 | Dockerfiles, Compose completo, CI, arquitectura | ⏳ |
| 9 | Metas de ahorro | ⏳ |
| 10 | Accesibilidad, rendimiento, seguridad y guía de despliegue | ⏳ |

## Licencia

[MIT](LICENSE).
