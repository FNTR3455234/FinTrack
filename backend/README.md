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
| `internal/modelos` | Documentos de MongoDB y DTOs de entrada/salida |
| `internal/repositorios` | Único código que consulta MongoDB |
| `internal/servicios` | Reglas de negocio y emisión de tokens |

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

🔒 = requiere `Authorization: Bearer <token de acceso>`.

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/api/v1/health` | Estado del servicio, ping a MongoDB y versión |
| POST | `/api/v1/auth/registro` | Alta de usuario. Devuelve la sesión iniciada |
| POST | `/api/v1/auth/login` | Inicio de sesión |
| POST | `/api/v1/auth/refresh` | Renueva el token de acceso |
| 🔒 GET | `/api/v1/auth/perfil` | Datos del usuario del token |
| 🔒 PUT | `/api/v1/auth/perfil` | Edita nombre y moneda |
| 🔒 | `/api/v1/cuentas`, `/api/v1/cuentas/:id` | CRUD de cuentas |
| 🔒 | `/api/v1/categorias`, `/api/v1/categorias/:id` | CRUD de categorías |
| 🔒 | `/api/v1/transacciones`, `/api/v1/transacciones/:id` | CRUD de transacciones con filtros y paginación |

El resto de los endpoints llega en las fases 5 y 6.

### `GET /health`

```bash
curl http://localhost:8080/api/v1/health
```

```json
{"datos":{"estado":"ok","mongo":"ok","version":"0.1.0","hora":"2026-08-01T05:04:57Z"}}
```

Responde `200` si MongoDB contesta y `503` con `"estado":"degradado"` si no, para que un
orquestador pueda dejar de mandarle tráfico.

### `POST /auth/registro`

```bash
curl -X POST http://localhost:8080/api/v1/auth/registro \
  -H "Content-Type: application/json" \
  -d '{"nombre":"Ana Lopez","email":"ana@fintrack.mx","password":"Clave12345!"}'
```

```json
{"datos":{
  "token_acceso":"eyJhbGciOi...",
  "token_refresco":"eyJhbGciOi...",
  "expira_en":900,
  "usuario":{"id":"...","nombre":"Ana Lopez","email":"ana@fintrack.mx","moneda":"MXN","activo":true}
}}
```

| Campo | Reglas |
|---|---|
| `nombre` | obligatorio, 2 a 80 caracteres |
| `email` | obligatorio, formato de correo, máx. 120 |
| `password` | obligatorio, 8 a **72** caracteres |
| `moneda` | opcional, 3 letras (por defecto `MXN`) |

El máximo de 72 no es arbitrario: bcrypt solo toma los primeros 72 bytes, así que aceptar más
daría una falsa sensación de seguridad.

Errores: `400 DATOS_INVALIDOS` (con la lista de campos), `400 JSON_INVALIDO`,
`409 EMAIL_YA_REGISTRADO`.

### `POST /auth/login`

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@fintrack.mx","password":"Demo1234!"}'
```

Devuelve lo mismo que el registro. Si el correo no existe **o** la contraseña es incorrecta,
responde exactamente el mismo `401 CREDENCIALES_INVALIDAS`: distinguirlos permitiría averiguar
qué correos tienen cuenta. Por la misma razón, cuando el correo no existe igual se ejecuta una
comparación bcrypt contra un hash de relleno, para que el tiempo de respuesta no lo delate.

### `POST /auth/refresh`

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"token_refresco":"eyJhbGciOi..."}'
```

```json
{"datos":{"token_acceso":"eyJhbGciOi...","expira_en":900}}
```

Vuelve a leer el usuario de la base antes de emitir el token: el de refresco dura 7 días y la
cuenta pudo borrarse o desactivarse en ese tiempo.

Errores: `401 TOKEN_VENCIDO`, `401 TOKEN_INVALIDO`, `403 CUENTA_DESACTIVADA`.

### `GET` y `PUT /auth/perfil`

```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/v1/auth/perfil

curl -X PUT -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"nombre":"Ana Lopez","moneda":"USD"}' \
  http://localhost:8080/api/v1/auth/perfil
```

El correo **no** se puede cambiar: es la credencial de acceso y la llave única.

## CRUD de cuentas, categorías y transacciones

Los tres recursos siguen la misma forma: `GET` lista, `POST` crea (201), `GET /:id` lee,
`PUT /:id` reemplaza y `DELETE /:id` borra (204).

### Cuentas

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"nombre":"BBVA Debito","tipo":"debito","saldo_inicial":18000,"color":"#2563EB"}' \
  http://localhost:8080/api/v1/cuentas
```

| Campo | Reglas |
|---|---|
| `nombre` | obligatorio, 1 a 60 caracteres |
| `tipo` | `efectivo`, `debito`, `credito` o `ahorro` |
| `saldo_inicial` | obligatorio; acepta 0 y negativos (tarjeta de crédito) |
| `color` | obligatorio, hexadecimal de 7 caracteres (`#2563EB`) |
| `archivada` | opcional |

`GET /cuentas` devuelve solo las activas; con `?incluir_archivadas=true` salen todas.

### Categorías

Mismos campos más `icono`, y `tipo` es `ingreso` o `gasto`. `GET /categorias` acepta
`?tipo=gasto` y `?incluir_archivadas=true`.

**No se puede cambiar el tipo de una categoría que ya tiene movimientos** (`409`): dejaría gastos
colgando de una categoría de ingreso y los reportes darían números sin sentido.

### Transacciones

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"cuenta_id":"...","categoria_id":"...","tipo":"gasto","monto":850.50,
       "descripcion":"Despensa","fecha":"2026-07-03T12:00:00Z","notas":"con cupon"}' \
  http://localhost:8080/api/v1/transacciones
```

El `monto` siempre es positivo: lo que decide si suma o resta es el `tipo`. Se redondea a dos
decimales al guardar.

**El `tipo` del movimiento tiene que coincidir con el de su categoría** (`400 TIPO_NO_COINCIDE`).
El tipo está duplicado a propósito —así el reporte de gastos por categoría filtra sin resolver la
categoría de cada documento— y esta es la única puerta por donde entra.

#### Filtros del listado

`GET /transacciones` acepta:

| Parámetro | Valores | Notas |
|---|---|---|
| `desde`, `hasta` | `AAAA-MM-DD` | `hasta` incluye el día completo, no corta a las 00:00 |
| `tipo` | `ingreso`, `gasto` | |
| `categoria_id`, `cuenta_id` | ObjectID | |
| `busqueda` | texto | Busca en descripción y notas, sin distinguir mayúsculas |
| `pagina` | ≥ 1 | Por defecto 1 |
| `limite` | 1 a 100 | Por defecto 20 |
| `orden` | `fecha_desc` (por defecto), `fecha_asc`, `monto_desc`, `monto_asc` | |

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/transacciones?desde=2026-07-01&hasta=2026-07-31&tipo=gasto&limite=5"
```

```json
{ "datos": [ ... ], "meta": { "pagina": 1, "limite": 5, "total": 17, "total_paginas": 4 } }
```

Los valores de paginación fuera de rango se ajustan (`pagina=0` pasa a 1, `limite=500` a 100),
pero una fecha o un identificador mal escritos sí devuelven `400`: ignorarlos en silencio daría
un listado que no es el que se pidió.

El texto de `busqueda` se escapa antes de armar la expresión regular, así que buscar `.*` no
recorre toda la colección.

### Borrado

`DELETE` de una cuenta o categoría **con movimientos** responde `409`
(`CUENTA_CON_TRANSACCIONES` / `CATEGORIA_CON_TRANSACCIONES`). No hay borrado en cascada: perder
los movimientos por equivocación no tiene vuelta atrás. Para eso existe `archivada`.

## Autenticación

Dos tokens, firmados con **secretos distintos**:

| Token | Vida | Para qué |
|---|---|---|
| Acceso | 15 min (`JWT_MINUTOS_ACCESO`) | Acompaña cada petición en `Authorization: Bearer` |
| Refresco | 7 días (`JWT_DIAS_REFRESCO`) | Solo sirve para pedir un token de acceso nuevo |

Usar secretos distintos hace que un token de refresco no pueda pasar por uno de acceso ni al
revés, aunque alguien lo mande en el encabezado equivocado. Además, cada token lleva dentro un
campo `tipo` que se verifica, y la validación exige el algoritmo HMAC (así un token con
`alg: none` se rechaza) y el emisor `fintrack`.

El middleware de autenticación mete el `usuario_id` del token en el contexto de Gin. **Los
handlers lo toman siempre de ahí y nunca del cuerpo ni de la query**: si el id viniera del
cliente, cualquiera podría pedir los datos de otro cambiando un valor. Esta es la regla que
sostiene todo el aislamiento entre usuarios.

El middleware distingue `TOKEN_VENCIDO` de `TOKEN_INVALIDO` para que el frontend sepa cuándo
vale la pena intentar el refresco y cuándo mandar directo al login.

### Contraseñas

Se guardan con **bcrypt** (coste 10, el de por defecto). El campo lleva `json:"-"`, así que el
hash nunca sale en una respuesta aunque se devuelva el usuario completo.

### Límite de peticiones

El grupo `/auth` está limitado a **20 peticiones por minuto y por IP** (contador de ventana fija
en memoria). Al pasarse responde `429 DEMASIADOS_INTENTOS` con el encabezado `Retry-After`.
Es la protección contra alguien probando contraseñas a fuerza bruta. El resto de la API no está
limitada.

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
make test               # unitarias; las de integracion se saltan solas
make test-integracion   # levanta MongoDB y corre TODO
```

En Windows: `.\make.ps1 test` y `.\make.ps1 test-integracion`.

Hay dos clases de prueba:

- **Unitarias.** Los servicios se prueban con repositorios en memoria que también filtran por
  `usuario_id`, para que una prueba de aislamiento no pase por accidente.
- **De integración.** Necesitan MongoDB y se **saltan solas** si no está la variable
  `MONGO_URI_PRUEBAS`, para que `go test ./...` funcione en cualquier máquina. Usan bases
  aparte (`fintrack_pruebas_*`) que se borran al terminar, así que nunca tocan los datos de
  desarrollo.

Las de `internal/rutas` levantan la **API completa contra MongoDB de verdad** —router,
middlewares, handlers, servicios y repositorios, sin ningún doble— y cubren el CRUD de los tres
recursos, los filtros, la paginación y la comprobación de que **dos usuarios nunca ven ni tocan
los datos del otro**.

Para correr solo esas:

```bash
make up
MONGO_URI_PRUEBAS="mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin" \
  go test ./internal/rutas/... -v
```

**Cobertura total: 83.2 %** (`go tool cover -func=coverage.out`). Por paquete, con las pruebas
de integración corriendo:

| Paquete | Cobertura |
|---|---|
| `internal/errores` · `internal/respuestas` · `internal/rutas` | 100 % |
| `internal/middleware` | 99.0 % |
| `internal/config` | 93.5 % |
| `internal/db` | 90.0 % |
| `internal/servicios` | 85.6 % |

`internal/handlers` y `internal/repositorios` dan un número bajo medidos por separado porque casi
todo su código lo ejercitan las pruebas de `internal/rutas`; por eso el total se mide con
`-coverpkg=./...`, que sí cuenta esa cobertura cruzada.
