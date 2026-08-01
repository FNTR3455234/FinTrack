# Bitácora de decisiones técnicas

Registro corto de las decisiones que no son obvias al leer el código, para poder
justificarlas después. Formato por entrada: **decisión, contexto, alternativas, por qué**.

---

## 001 — El dinero se guarda como `float64` en pesos

**Fecha:** 2026-07-31 · **Fase:** 0

- **Contexto:** `monto`, `saldo_inicial` y `monto_limite` necesitan un tipo en Go y en MongoDB.
- **Alternativas:** `int64` en centavos (exacto, pero obliga a convertir en la semilla, el CSV,
  los formularios y las gráficas); `Decimal128` de Mongo (el tipo correcto para dinero, pero Go
  no puede operar aritméticamente con él sin pasar por `string`/`big.Float`).
- **Decisión:** `float64` en pesos. En Mongo, Go y JSON se ve igual: `"monto": 1250.50`.
- **Por qué:** con el volumen de este proyecto (cientos de transacciones, sumas de dos decimales)
  el error de punto flotante nunca alcanza el centavo, y el código queda explicable línea por
  línea sin conversiones en cada capa. Los totales de los reportes se redondean a 2 decimales al
  devolverse.

---

## 002 — `Makefile` y además `make.ps1`

**Fecha:** 2026-07-31 · **Fase:** 0

- **Contexto:** el desarrollo es en Windows, donde `make` no viene instalado; el CI corre en
  Ubuntu y la rúbrica pide un Makefile.
- **Alternativas:** solo `Makefile` e instalar `make` con winget; solo un script de PowerShell.
- **Decisión:** los dos, con los mismos targets y los mismos comandos.
- **Por qué:** el `Makefile` es el artefacto que se entrega y el que usa el CI; `make.ps1` evita
  depender de una instalación manual para trabajar en el día a día. Si se cambia un target hay
  que cambiarlo en ambos archivos.

---

## 003 — MongoDB standalone, sin replica set

**Fecha:** 2026-07-31 · **Fase:** 0

- **Contexto:** las transacciones multi-documento de MongoDB requieren un replica set, aunque sea
  de un solo nodo.
- **Alternativas:** configurar un replica set de un nodo en Compose desde el inicio.
- **Decisión:** `mongo:7` standalone.
- **Por qué:** ninguna operación del diseño necesita atomicidad entre varios documentos: cada
  escritura toca una sola colección. Añadir el replica set complicaría el arranque (`rs.initiate`,
  esperas) sin ganar nada. Si más adelante hace falta, se cambia aquí y se anota.

---

## 004 — Nombres en español en JSON y en MongoDB

**Fecha:** 2026-07-31 · **Fase:** 0

- **Contexto:** el código Go usa identificadores exportados en inglés por convención del lenguaje,
  pero los campos de la API y de la base son parte del entregable.
- **Decisión:** campos JSON y BSON en español (`monto`, `fecha`, `categoria_id`), en `snake_case`;
  comentarios, mensajes de error y commits también en español.
- **Por qué:** el proyecto se defiende en español y los códigos de error (`CATEGORIA_NO_ENCONTRADA`)
  se leen igual en la API, en Postman y en el frontend.

---

## 005 — Esquema estricto y `Double()` explícito en la semilla

**Fecha:** 2026-07-31 · **Fase:** 1

- **Contexto:** el validador `$jsonSchema` exige `bsonType: "double"` para el dinero, pero la
  primera carga de la semilla falló con `Document failed validation`.
- **Causa:** `mongosh` guarda los números enteros de JavaScript como **int32** (a diferencia del
  shell antiguo, que usaba double siempre), así que `saldo_inicial: 2500` llegaba como int.
- **Alternativas:** aflojar el esquema a `["double", "int", "long"]`.
- **Decisión:** mantener el esquema estricto en `double` y envolver los montos de la semilla con
  `Double()` desde la función `pesos()`.
- **Por qué:** `double` es exactamente lo que escribe Go (`float64`), que es el único cliente real.
  Aflojar el tipo habría escondido el problema y permitido que convivieran dos representaciones
  del dinero en la misma colección. Los campos `mes` y `anio` sí aceptan `["int", "long"]` porque
  Go puede mandar int32 o int64 según el valor.

---

## 006 — La semilla es determinista y está anclada a una fecha fija

**Fecha:** 2026-07-31 · **Fase:** 1

- **Contexto:** los 120 movimientos de ejemplo se reparten en 6 meses.
- **Alternativas:** generar montos aleatorios y fechas relativas a "hoy".
- **Decisión:** `_id` fijos, montos calculados con una lista fija de factores de variación y
  fechas ancladas a las constantes `ANIO_FINAL` / `MES_FINAL` (julio de 2026).
- **Por qué:** las pruebas de la fase 5 tienen que poder esperar totales exactos de las dos
  agregaciones. Con datos aleatorios habría que aflojar las aserciones. El costo es que la
  semilla envejece: se re-ancla cambiando dos constantes y volviendo a correr `make seed`.

---

## 007 — `$setWindowFields` para el porcentaje de gasto por categoría

**Fecha:** 2026-07-31 · **Fase:** 1

- **Contexto:** el reporte de gastos por categoría necesita el porcentaje de cada categoría sobre
  el gasto total del periodo, y el total solo se conoce después del `$group`.
- **Alternativas:** una segunda consulta para el total, o un `$group` de todo en un solo documento
  seguido de `$unwind`.
- **Decisión:** una etapa `$setWindowFields` con ventana `["unbounded", "unbounded"]`.
- **Por qué:** resuelve el total en la misma consulta, sin ida y vuelta extra a la base y sin
  desarmar y rearmar el arreglo de resultados. Es una etapa y se explica en una línea.

---

## 008 — El módulo declara `go 1.25`, no `go 1.22`

**Fecha:** 2026-07-31 · **Fase:** 2

- **Contexto:** el plan pedía Go 1.22+, pero `go mod tidy` subió la directiva a `go 1.25.0`.
- **Causa:** Gin 1.12 declara `go 1.25.0` y `golang.org/x/net` también; no es algo que se pueda
  bajar sin fijar versiones viejas de media docena de dependencias.
- **Alternativas:** fijar Gin en v1.10.1 (probado: sigue sin bastar, `x/net` lo exige igual).
- **Decisión:** dejar `go 1.25.0` y usar las versiones actuales de las dependencias.
- **Por qué:** 1.25 cumple de sobra el "1.22 o superior" del plan, y pelearse con el ecosistema
  para bajar un número en go.mod no aporta nada. El CI de la fase 8 fijará Go 1.25.

---

## 009 — Un paquete `respuestas` aparte, fuera de `handlers`

**Fecha:** 2026-07-31 · **Fase:** 2

- **Contexto:** la estructura del plan no incluía un paquete para el formato uniforme de
  respuesta.
- **Alternativas:** ponerlo dentro de `handlers`.
- **Decisión:** `internal/respuestas`, con `OK`, `Creado`, `Paginado`, `SinContenido` y `Fallo`.
- **Por qué:** los middlewares también tienen que responder con el mismo formato (el de
  autenticación devuelve 401, el de recuperación 500). Si viviera en `handlers`, `middleware`
  tendría que importar `handlers`, que es justo al revés del flujo de dependencias.

---

## 010 — Middlewares escritos a mano en vez de librerías

**Fecha:** 2026-07-31 · **Fase:** 2

- **Contexto:** hay librerías conocidas para CORS (`gin-contrib/cors`) y bitácora.
- **Decisión:** escribir CORS, bitácora, id de petición y recuperación a mano.
- **Por qué:** son entre 30 y 60 líneas cada uno, se explican completos en la revisión oral y
  ahorran cuatro dependencias. La rúbrica además pide middlewares propios.

---

## Pendientes anotados (se resuelven en su fase)

- **Fase 3 — Refresh token sin estado:** el refresh token no se guarda en la base, así que se puede
  renovar pero no revocar antes de sus 7 días. Aceptable para el alcance; se documentará como
  limitación conocida en vez de añadir una colección de sesiones.
- **Fase 4 — `tipo` duplicado:** una transacción guarda `tipo` y su categoría también. Es
  deliberado (la agregación de gastos por categoría filtra sin `$lookup` previo), pero el servicio
  debe validar que `transaccion.tipo == categoria.tipo`.
- **Fase 4 — Borrado de cuentas y categorías:** si tienen transacciones asociadas, `DELETE`
  responde `409` y el cliente debe archivar (`archivada: true`) en lugar de borrar. Sin cascada.
- **Fase 5 — Zona horaria:** las fechas se guardan en UTC. Decidir si los rangos de mes se calculan
  en UTC o en `America/Mexico_City` (afecta a las transacciones de fin de mes por la noche).
- **Fase 5 — Saldo de cuentas:** no se guarda un saldo actual, solo `saldo_inicial`; el saldo se
  calcula agregando transacciones para no tener dos fuentes de verdad.
