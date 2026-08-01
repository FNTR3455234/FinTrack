# Entregable 1 — Base de datos

Base de datos **MongoDB 7** de FinTrack: colecciones con validación de esquema, índices, datos
semilla y respaldo.

- Motor: **MongoDB 7.0** (imagen oficial `mongo:7`)
- Base: `fintrack`
- Colecciones: `usuarios`, `cuentas`, `categorias`, `transacciones`, `presupuestos`
- Modelo y diagrama: [`modelo.md`](modelo.md)

## Archivos

| Archivo | Qué hace |
|---|---|
| [`01_crear_colecciones.js`](01_crear_colecciones.js) | Crea las 5 colecciones con su validador `$jsonSchema` y los 6 índices. Idempotente. |
| [`02_insertar_datos.js`](02_insertar_datos.js) | Carga el usuario demo con 3 cuentas, 10 categorías, 120 transacciones y 6 presupuestos. Idempotente. |
| [`03_respaldo.sh`](03_respaldo.sh) | Respalda y restaura la base con `mongodump`/`mongorestore`. |
| [`modelo.md`](modelo.md) | Diagrama entidad-relación, relaciones, índices y decisiones de modelado. |

## Herramientas empleadas

| Herramienta | Versión | Para qué |
|---|---|---|
| MongoDB Server | 7.0 | Motor de base de datos |
| `mongosh` | 2.x | Shell con el que se ejecutan los scripts `.js` (viene dentro de la imagen `mongo:7`) |
| `mongodump` / `mongorestore` | Database Tools 100.x | Respaldo y restauración (también dentro de la imagen) |
| Docker Compose | v2+ | Levanta el servidor y monta los scripts |

No hace falta instalar nada de MongoDB en la máquina: todo corre dentro del contenedor.

## Instalación y ejecución

**Requisito único:** Docker Desktop instalado y corriendo.

```bash
# Desde la raiz del repositorio
make up          # Windows sin make:  .\make.ps1 up
```

Eso levanta `mongo:7` en `localhost:27017` y, **la primera vez**, ejecuta solo los scripts
`01` y `02` (Docker corre lo que hay en `/docker-entrypoint-initdb.d` cuando el volumen está
vacío). No hay pasos manuales.

Para volver a aplicar el esquema y recargar los datos en cualquier momento:

```bash
make seed        # Windows:  .\make.ps1 seed
```

Salida esperada:

```
Definiendo colecciones en la base 'fintrack'...
  coleccion actualizada: usuarios
  coleccion actualizada: cuentas
  coleccion actualizada: categorias
  coleccion actualizada: transacciones
  coleccion actualizada: presupuestos
Creando indices...
Listo. Colecciones e indices de FinTrack en su lugar.
Limpiando datos previos del usuario demo...
Semilla cargada para demo@fintrack.mx (clave: Demo1234!):
  cuentas:       3
  categorias:    10
  transacciones: 120
  presupuestos:  6
```

Los dos scripts son **idempotentes**: `01` usa `collMod` si la colección ya existe y `createIndex`
(que no hace nada si el índice ya está); `02` borra primero todo lo del usuario demo y lo vuelve a
insertar. Correrlos diez veces deja exactamente el mismo resultado que correrlos una.

### Conectarse a mano

```bash
docker compose -f docker-compose.dev.yml exec mongo \
  mongosh -u fintrack_admin -p fintrack_dev_2026 --authenticationDatabase admin fintrack
```

Cadena de conexión para Compass o para el backend:

```
mongodb://fintrack_admin:fintrack_dev_2026@localhost:27017/?authSource=admin
```

## Datos semilla

Un usuario demo con **6 meses de historia** (febrero–julio de 2026), 20 movimientos por mes:

| Elemento | Cantidad | Detalle |
|---|---|---|
| Usuario | 1 | `demo@fintrack.mx` / `Demo1234!` (la contraseña se guarda como hash bcrypt) |
| Cuentas | 3 | Efectivo, BBVA Débito, Santander Crédito (saldo inicial negativo) |
| Categorías | 10 | 2 de ingreso (Nómina, Freelance) y 8 de gasto |
| Transacciones | 120 | 20 por mes × 6 meses, montos en pesos mexicanos |
| Presupuestos | 6 | Del mes de julio de 2026 |

Resumen mensual que produce la semilla:

| Mes | Movimientos | Ingresos | Gastos | Balance |
|---|---|---|---|---|
| 2026-02 | 20 | 28,325.00 | 19,407.00 | 8,918.00 |
| 2026-03 | 20 | 28,780.00 | 20,475.60 | 8,304.40 |
| 2026-04 | 20 | 28,150.00 | 18,996.00 | 9,154.00 |
| 2026-05 | 20 | 28,920.00 | 20,804.40 | 8,115.60 |
| 2026-06 | 20 | 28,605.00 | 20,064.60 | 8,540.40 |
| 2026-07 | 20 | 28,710.00 | 20,311.20 | 8,398.80 |

Los `_id` son **fijos** (`650000...01` el usuario, `650100...` las cuentas, `650200...` las
categorías) y los montos **no son aleatorios**: cada mes aplica un factor de variación tomado de
una lista fija. Así la semilla es reproducible y las pruebas automáticas pueden esperar totales
exactos.

Los presupuestos de julio están elegidos a propósito para que el reporte muestre los **tres
estados** posibles: uno excedido, dos en alerta y tres en orden.

> **Para re-anclar las fechas:** cambia `ANIO_FINAL` y `MES_FINAL` al principio de
> `02_insertar_datos.js` y vuelve a correr `make seed`. Los 6 meses se recalculan solos.

## Las dos consultas relacionales

Las dos usan `$lookup` para cruzar colecciones. El backend las ejecuta en
`/api/v1/reportes/gastos-por-categoria` y `/api/v1/reportes/estado-presupuestos` (fase 5); aquí
están en su versión de `mongosh` para poder probarlas directamente contra la semilla.

### 1. Gastos por categoría

**Objetivo:** identificar en qué categorías se concentra el gasto de un periodo, con el total, el
número de movimientos y el peso porcentual de cada una sobre el gasto total. Responde a la
pregunta *"¿en qué se me está yendo el dinero?"*, que es la que ordena el dashboard.

**Colecciones que cruza:** `transacciones` → `categorias`.

```js
const usuario = ObjectId("650000000000000000000001");
const desde = new Date(Date.UTC(2026, 6, 1));   // 1 de julio de 2026
const hasta = new Date(Date.UTC(2026, 7, 1));   // 1 de agosto de 2026

db.transacciones.aggregate([
  // 1. Solo los gastos de ESTE usuario dentro del periodo. Va primero para que
  //    el indice (usuario_id, fecha) recorte el conjunto cuanto antes.
  { $match: { usuario_id: usuario, tipo: "gasto", fecha: { $gte: desde, $lt: hasta } } },

  // 2. Se acumula por categoria: cuanto y cuantas veces.
  { $group: { _id: "$categoria_id", total: { $sum: "$monto" }, cantidad: { $sum: 1 } } },

  // 3. Se trae el nombre y el color de la categoria (la parte "relacional").
  { $lookup: { from: "categorias", localField: "_id", foreignField: "_id", as: "categoria" } },
  { $unwind: "$categoria" },

  // 4. Suma de TODOS los grupos, para poder sacar el porcentaje sin una segunda consulta.
  { $setWindowFields: {
      output: { gran_total: { $sum: "$total", window: { documents: ["unbounded", "unbounded"] } } },
  } },

  // 5. Se arma la respuesta final.
  { $project: {
      _id: 0,
      categoria_id: "$_id",
      nombre: "$categoria.nombre",
      color: "$categoria.color",
      total: { $round: ["$total", 2] },
      cantidad: 1,
      porcentaje: { $round: [{ $multiply: [{ $divide: ["$total", "$gran_total"] }, 100] }, 2] },
  } },

  { $sort: { total: -1 } },
]);
```

Resultado real contra la semilla (julio de 2026):

| Categoría | Total | Movimientos | % del gasto |
|---|---|---|---|
| Renta | 9,500.00 | 1 | 46.77 % |
| Supermercado | 4,118.10 | 4 | 20.28 % |
| Servicios | 1,680.20 | 3 | 8.27 % |
| Restaurantes | 1,319.70 | 3 | 6.50 % |
| Educación | 1,200.00 | 1 | 5.91 % |
| Salud | 901.00 | 1 | 4.44 % |
| Transporte | 816.20 | 2 | 4.02 % |
| Entretenimiento | 776.00 | 2 | 3.82 % |
| **Total** | **20,311.20** | **17** | |

> Los porcentajes suman 100.01 % porque cada uno se redondea a dos decimales por separado. Es un
> artefacto del redondeo, no un error de la suma.

### 2. Estado de presupuestos

**Objetivo:** comparar lo presupuestado contra lo realmente gastado en cada categoría durante un
mes, y marcar cuáles están en orden, cuáles cerca del límite y cuáles ya se pasaron. Es lo que
alimenta las barras de color del dashboard y las alertas al registrar un gasto.

**Colecciones que cruza:** `presupuestos` → `categorias` y `presupuestos` → `transacciones`.

El segundo `$lookup` usa la forma con `let` + `pipeline` porque la condición no es una simple
igualdad de campos: hay que cruzar por usuario **y** categoría, y además filtrar por tipo y por
rango de fechas dentro de la colección relacionada.

```js
db.presupuestos.aggregate([
  // 1. Los presupuestos de este usuario para el mes pedido.
  { $match: { usuario_id: usuario, mes: 7, anio: 2026 } },

  // 2. Nombre y color de la categoria presupuestada.
  { $lookup: { from: "categorias", localField: "categoria_id", foreignField: "_id", as: "categoria" } },
  { $unwind: "$categoria" },

  // 3. Lo realmente gastado en esa categoria durante el mes.
  //    Con let + pipeline se puede filtrar por varias condiciones a la vez.
  { $lookup: {
      from: "transacciones",
      let: { cat: "$categoria_id", usr: "$usuario_id" },
      pipeline: [
        { $match: { $expr: { $and: [
          { $eq: ["$usuario_id", "$$usr"] },
          { $eq: ["$categoria_id", "$$cat"] },
          { $eq: ["$tipo", "gasto"] },
          { $gte: ["$fecha", desde] },
          { $lt: ["$fecha", hasta] },
        ] } } },
        { $group: { _id: null, gastado: { $sum: "$monto" } } },
      ],
      as: "movimientos",
  } },

  // 4. Si no hubo ni un gasto, el array viene vacio: se toma 0.
  { $addFields: { gastado: { $round: [{ $ifNull: [{ $first: "$movimientos.gastado" }, 0] }, 2] } } },

  { $addFields: {
      disponible: { $round: [{ $subtract: ["$monto_limite", "$gastado"] }, 2] },
      porcentaje_usado: { $round: [{ $multiply: [{ $divide: ["$gastado", "$monto_limite"] }, 100] }, 2] },
  } },

  // 5. El semaforo: ok < 80 %, alerta 80-100 %, excedido > 100 %.
  { $addFields: { estado: { $switch: { branches: [
      { case: { $gt: ["$porcentaje_usado", 100] }, then: "excedido" },
      { case: { $gte: ["$porcentaje_usado", 80] }, then: "alerta" },
  ], default: "ok" } } } },

  { $project: { _id: 0, categoria_id: 1, nombre: "$categoria.nombre",
                monto_limite: 1, gastado: 1, disponible: 1, porcentaje_usado: 1, estado: 1 } },
  { $sort: { porcentaje_usado: -1 } },
]);
```

Resultado real contra la semilla (julio de 2026):

| Categoría | Límite | Gastado | Disponible | % usado | Estado |
|---|---|---|---|---|---|
| Supermercado | 4,000.00 | 4,118.10 | −118.10 | 102.95 % | 🔴 excedido |
| Restaurantes | 1,500.00 | 1,319.70 | 180.30 | 87.98 % | 🟡 alerta |
| Servicios | 2,000.00 | 1,680.20 | 319.80 | 84.01 % | 🟡 alerta |
| Entretenimiento | 1,000.00 | 776.00 | 224.00 | 77.60 % | 🟢 ok |
| Transporte | 1,200.00 | 816.20 | 383.80 | 68.02 % | 🟢 ok |
| Salud | 1,500.00 | 901.00 | 599.00 | 60.07 % | 🟢 ok |

## Respaldo y restauración

```bash
# Crear un respaldo (queda en respaldos/, que esta en .gitignore)
./database/03_respaldo.sh respaldar

# Ver los respaldos existentes
./database/03_respaldo.sh listar

# Restaurar (--drop: reemplaza el contenido actual)
./database/03_respaldo.sh restaurar respaldos/fintrack_20260731_2215.archive.gz
```

Usa `mongodump --archive --gzip` dentro del contenedor y guarda el resultado en un único archivo
comprimido en la máquina anfitriona, así que no depende de copiar carpetas desde el contenedor ni
de tener las Database Tools instaladas.

En Windows se ejecuta con Git Bash: `bash database/03_respaldo.sh respaldar`.

## Verificación rápida

```bash
docker compose -f docker-compose.dev.yml exec mongo \
  mongosh -u fintrack_admin -p fintrack_dev_2026 --authenticationDatabase admin --quiet fintrack \
  --eval 'db.getCollectionNames().forEach(c => print(c + ": " + db[c].countDocuments()))'
```

```
categorias: 10
cuentas: 3
presupuestos: 6
transacciones: 120
usuarios: 1
```
