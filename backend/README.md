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
| [`github.com/golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) | Firma y validación de los tokens |
| [`golang.org/x/crypto/bcrypt`](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | Hash de contraseñas |
| [`github.com/go-playground/validator/v10`](https://github.com/go-playground/validator) | Validación de los cuerpos de petición |
| [`github.com/swaggo/swag`](https://github.com/swaggo/swag) + [`gin-swagger`](https://github.com/swaggo/gin-swagger) | Genera y sirve la especificación OpenAPI |
| [`github.com/stretchr/testify`](https://github.com/stretchr/testify) | Aserciones en las pruebas |
| `log/slog` (biblioteca estándar) | Bitácora estructurada |
| `encoding/csv` (biblioteca estándar) | Exportación e importación de movimientos |

No hay ninguna otra: los middlewares (bitácora, recuperación, CORS, límite de peticiones) están
escritos a mano ([decisión 010](../docs/decisiones.md)).

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
level=INFO msg="iniciando FinTrack" version=0.1.0-dev modo=debug
level=INFO msg="conectado a MongoDB" base=fintrack
level=INFO msg="indices verificados" cantidad=6
level=INFO msg="servidor escuchando" direccion=:8080
```

### En contenedor

`backend/Dockerfile` es multi-etapa: `golang:1.25-alpine` compila y `alpine:3.21` ejecuta. En la
imagen final —unos 64 MB— no hay compilador ni código fuente, y el proceso corre como el usuario
`fintrack` (uid 10001), no como root.

```bash
docker build -t fintrack-api --build-arg VERSION=1.2.3 ./backend
```

`VERSION` se graba en el binario con `-ldflags -X .../config.Version`, así que `/api/v1/health`
devuelve de qué compilación salió lo que está corriendo. Sin la bandera queda `0.1.0-dev`.

El binario se compila con `CGO_ENABLED=0`, así que es estático y no depende de la libc; de Alpine
solo se usan los certificados y `wget`, que es lo que ejecuta el `HEALTHCHECK` contra
`/api/v1/health`. Ese healthcheck comprueba también que MongoDB responde, por eso Compose lo usa
para no arrancar el frontend antes de tiempo.

Lo normal no es construir la imagen a mano: `make arriba` desde la raíz levanta el stack completo
(ver [`docs/arquitectura.md`](../docs/arquitectura.md)).

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
| 🔒 | `/api/v1/presupuestos`, `/api/v1/presupuestos/:id` | CRUD de presupuestos mensuales |
| 🔒 GET | `/api/v1/reportes/gastos-por-categoria` | **Consulta relacional 1**: en qué se fue el dinero |
| 🔒 GET | `/api/v1/reportes/estado-presupuestos` | **Consulta relacional 2**: presupuestado contra gastado |
| 🔒 GET | `/api/v1/reportes/resumen` | Cifras del mes para el tablero |
| 🔒 GET | `/api/v1/reportes/tendencia` | Serie mensual de ingresos y gastos |
| 🔒 GET | `/api/v1/reportes/saldos` | Saldo actual de cada cuenta |
| 🔒 GET | `/api/v1/transacciones/exportar` | Descarga los movimientos como CSV |
| 🔒 POST | `/api/v1/transacciones/importar` | Carga movimientos desde un CSV |
| GET | `/swagger` | Documentación interactiva (OpenAPI) |

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

#### La fecha es un día, no un instante

El `fecha` que llega se guarda como las **12:00 UTC del día que eligió el usuario**, leyendo el día
en el huso con el que vino. Un gasto enviado como `2026-07-31T19:00:00-06:00` queda guardado como
`2026-07-31T12:00:00Z` y cuenta contra julio, no contra agosto. Nadie apunta un gasto "a las
19:03:22": lo apunta el día 31 ([decisión 019](../docs/decisiones.md)).

### Borrado

`DELETE` de una cuenta o categoría **con movimientos** responde `409`
(`CUENTA_CON_TRANSACCIONES` / `CATEGORIA_CON_TRANSACCIONES`). Una categoría **con presupuestos**
responde `409 CATEGORIA_CON_PRESUPUESTOS`. No hay borrado en cascada: perder los movimientos por
equivocación no tiene vuelta atrás. Para eso existe `archivada`.

## Presupuestos

Un presupuesto es el techo de gasto que el usuario se pone en una categoría para un mes concreto.

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"categoria_id":"...","monto_limite":4000,"mes":7,"anio":2026}' \
  http://localhost:8080/api/v1/presupuestos
```

| Campo | Reglas |
|---|---|
| `categoria_id` | ObjectID de una categoría **de gasto** del usuario |
| `monto_limite` | mayor que 0 |
| `mes` | 1 a 12 |
| `anio` | 2000 a 2100 |

`mes` y `anio` se guardan como enteros y no como fecha: el presupuesto es del periodo entero, no
de un instante.

- **Solo se presupuestan gastos** (`400 TIPO_NO_COINCIDE`): ponerle un techo a un ingreso no
  significa nada, y la consulta de estado solo suma transacciones de tipo `gasto`.
- **Uno por categoría y mes** (`409 PRESUPUESTO_DUPLICADO`). Lo decide el índice único
  `(usuario_id, categoria_id, mes, anio)`, no una consulta previa: entre la consulta y el insert
  cabe otra petición.
- `GET /presupuestos?mes=7&anio=2026` filtra por periodo; sin esos parámetros los devuelve todos.

### Alertas al registrar un gasto

`POST /transacciones` responde la transacción de siempre y, si el gasto deja su categoría en el
**80 % o más** de su presupuesto del mes, añade un campo `alerta`:

```json
{
  "datos": {
    "id": "...", "tipo": "gasto", "monto": 250.75, "fecha": "2026-07-22T12:00:00Z",
    "alerta": {
      "nombre": "Supermercado",
      "monto_limite": 4000, "gastado": 4368.85, "disponible": -368.85,
      "porcentaje_usado": 109.22, "estado": "excedido",
      "mensaje": "Te pasaste del presupuesto de Supermercado: llevas 4368.85 de 4000.00 (109.22%), 368.85 de mas."
    }
  }
}
```

El campo lleva `omitempty`: si no hay presupuesto, si el movimiento es un ingreso o si el gasto
todavía va holgado, ni aparece. La transacción se serializa igual que antes, así que un cliente de
la fase 4 sigue funcionando sin cambios.

Dos detalles del diseño:

- La alerta se calcula **después** de guardar, para que el total ya incluya el movimiento que el
  usuario acaba de capturar.
- Un fallo al calcularla **no tumba la petición**: la transacción ya está guardada y responder
  `500` haría creer que no se registró. Se anota en la bitácora y se responde sin alerta.

El semáforo lo resuelve la misma agregación que alimenta `/reportes/estado-presupuestos`, así que
la alerta y el tablero no pueden decir cosas distintas del mismo presupuesto.

## Reportes

Cinco consultas de solo lectura, todas bajo `/api/v1/reportes`. Aceptan `?mes=` y `?anio=`; sin
ellos usan el mes en curso. Un periodo imposible o mal escrito responde `400 PERIODO_INVALIDO` en
vez de devolver ceros, que se leerían como "no gastaste nada".

| Endpoint | Qué responde | Colecciones que cruza |
|---|---|---|
| `gastos-por-categoria` | Total, número de movimientos y % de cada categoría | `transacciones` → `categorias` |
| `estado-presupuestos` | Presupuestado, gastado, disponible y semáforo | `presupuestos` → `categorias` + `transacciones` |
| `resumen` | Ingresos, gastos, balance, saldo total y conteo del semáforo | las tres anteriores |
| `tendencia` | Serie mensual (`?meses=`, 1 a 24, por defecto 6) | `transacciones` |
| `saldos` | Saldo actual de cada cuenta | `cuentas` → `transacciones` |

Las dos primeras son **las consultas relacionales de la entrega**. Su objetivo está documentado
como comentario en `repositorios/reportes_gastos.go` y `repositorios/reportes_presupuestos.go`, y
su versión de `mongosh` con resultados reales en [`database/README.md`](../database/README.md).

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/reportes/gastos-por-categoria?mes=7&anio=2026"
```

Contra los datos semilla (julio de 2026):

| Categoría | Total | Movimientos | % del gasto |
|---|---|---|---|
| Renta | 9 500.00 | 1 | 46.77 % |
| Supermercado | 4 118.10 | 4 | 20.28 % |
| Servicios | 1 680.20 | 3 | 8.27 % |
| … | | | |

### El semáforo

| Estado | Cuándo |
|---|---|
| `ok` | menos del 80 % del límite |
| `alerta` | del 80 % al 100 % |
| `excedido` | más del 100 % |

Los umbrales viven en `modelos/reportes.go` y los aplica el `$switch` de la agregación, para que el
valor del código y el de la consulta no se puedan separar sin que se note.

### Tendencia

Los meses sin ningún movimiento **también salen, con ceros**. MongoDB solo agrupa lo que existe, así
que los huecos los rellena el servicio: una gráfica de barras a la que le faltan meses miente sobre
la forma de la serie.

### Saldos

`saldo_inicial + ingresos − gastos`, calculado, nunca guardado. Un saldo almacenado sería una
segunda fuente de verdad que se desincroniza en cuanto una transacción se edita o se borra
([decisión 020](../docs/decisiones.md)).

## Exportar e importar CSV

Las dos mitades encajan: **lo que exporta la API se puede volver a importar sin editarlo**.

```bash
# Exportar (acepta los mismos filtros del listado, sin paginar)
curl -H "Authorization: Bearer $TOKEN" -O -J \
  "http://localhost:8080/api/v1/transacciones/exportar?desde=2026-07-01&hasta=2026-07-31"

# Importar
curl -H "Authorization: Bearer $TOKEN" -F "archivo=@movimientos.csv" \
  http://localhost:8080/api/v1/transacciones/importar
```

Columnas: `fecha,tipo,cuenta,categoria,monto,descripcion,notas`

```csv
fecha,tipo,cuenta,categoria,monto,descripcion,notas
2026-07-03,gasto,BBVA Debito,Supermercado,850.50,Despensa,con cupon
2026-07-04,ingreso,BBVA Debito,Nomina,20000.00,Quincena,
```

Detalles que importan al abrirlo en Excel:

- La cuenta y la categoría van **por nombre**, no por identificador: un `ObjectID` de 24 caracteres
  no le sirve a nadie en una hoja de cálculo. Al importar se resuelven por nombre, sin distinguir
  mayúsculas ni espacios de sobra.
- El archivo empieza con la **marca BOM** de UTF-8, sin la cual Excel parte los acentos. La
  importación la quita al leer.
- El orden de las columnas del encabezado **da igual**, pero tienen que estar todas.
- Al importar se aceptan las comas de millar y el signo de pesos (`"$1,250.50"`), que es lo que
  escribe Excel al dar formato de moneda. Lo que **no** se acepta es un monto negativo: el signo lo
  decide el `tipo`.

### O entra el archivo completo o no entra nada

Si una sola fila falla, se responde `400 CSV_INVALIDO` y **no se guarda ninguna**:

```json
{
  "error": {
    "codigo": "CSV_INVALIDO",
    "mensaje": "El archivo tiene filas que no se pueden importar. No se guardo ninguna.",
    "detalles": [
      "fila 3: la cuenta \"Cuenta fantasma\" no existe",
      "fila 4: la fecha \"03/08/2026\" no tiene el formato AAAA-MM-DD"
    ]
  }
}
```

Se validan **todas** las filas antes de escribir nada, y se reportan todos los errores de una vez.
La razón es que MongoDB está en modo standalone y no hay transacciones multidocumento: insertar
sobre la marcha dejaría archivos importados por la mitad, y reintentar duplicaría lo que sí entró
([decisión 023](../docs/decisiones.md)). El número de fila es el que se ve en la hoja de cálculo.

Las reglas de cada fila son **exactamente** las de `POST /transacciones`: mismo rango de monto,
misma longitud de descripción, mismo cruce de tipo con la categoría. Una vía de entrada que valide
menos que la otra sería una puerta trasera al modelo de datos.

Límites: 2 MiB por archivo, 5 000 filas al importar y 10 000 al exportar.

## Documentación interactiva (Swagger)

Con la API corriendo: **[http://localhost:8080/swagger](http://localhost:8080/swagger)**.

Son **33 operaciones sobre 20 rutas**, con sus parámetros, sus cuerpos de ejemplo y los códigos de
error que puede devolver cada una. El botón **Authorize** acepta el token de acceso y a partir de
ahí se puede probar cualquier endpoint desde el navegador.

La especificación no se escribe a mano: la genera [swaggo](https://github.com/swaggo/swag) leyendo
las anotaciones que hay **sobre cada handler**.

```bash
make swagger    # Windows:  .\make.ps1 swagger
```

`backend/docs/` se versiona aunque sea código generado, porque `internal/rutas` lo importa por su
`init()`: sin ese directorio el proyecto no compila, y ignorarlo obligaría a instalar `swag` antes
de poder construir ([decisión 025](../docs/decisiones.md)).

`/swagger` va fuera de `/api/v1` y sin autenticación: describe la API, no expone datos. En un
despliegue público conviene dejarlo solo en entornos internos.

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
middlewares, handlers, servicios y repositorios, sin ningún doble— y cubren el CRUD de los cuatro
recursos, los filtros, la paginación, las cinco agregaciones y la comprobación de que **dos
usuarios nunca ven ni tocan los datos del otro**.

Las dos consultas relacionales se prueban contra un juego de datos chico y conocido, montado por la
propia prueba, en el que cada cifra que se afirma se puede sumar a mano: los tres estados del
semáforo, los porcentajes, el mes de al lado que no debe colarse y un presupuesto sin gastos que
tiene que aparecer en cero.

Para correr solo esas:

```bash
make up
MONGO_URI_PRUEBAS="mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin" \
  go test ./internal/rutas/... -v
```

**Cobertura total: 84.1 %**, con las pruebas de integración corriendo:

| Paquete | Cobertura |
|---|---|
| `internal/errores` · `internal/respuestas` · `internal/rutas` | 100 % |
| `internal/middleware` | 99.0 % |
| `internal/config` | 93.5 % |
| `internal/modelos` | 90.9 % |
| `internal/db` | 90.0 % |
| `internal/servicios` | 89.3 % |
| `internal/repositorios` | 85.2 % |
| `internal/handlers` | 79.1 % |

Medidos con `-coverpkg=./cmd/...,./internal/...`. Sin esa bandera, `handlers` y `repositorios`
darían un número mucho más bajo: casi todo su código lo ejercitan las pruebas de `internal/rutas`,
y por defecto Go solo cuenta la cobertura que produce el paquete que se está probando. `docs/` queda
fuera de la medición porque es código generado. Lo que baja de `cmd/api` (0 %) es el cableado y el
apagado ordenado, que se comprueban a mano mandando una señal real al binario.

Además, la [colección de Postman](../postman/README.md) corre **41 peticiones con 105 aserciones**
contra la API viva (`make postman`). Las pruebas de Go comprueban el comportamiento; la colección
comprueba que lo que la API dice de sí misma en Swagger coincide con lo que hace — de hecho así se
encontró que `/auth/refresh` estaba mal documentado ([decisión 026](../docs/decisiones.md)).
