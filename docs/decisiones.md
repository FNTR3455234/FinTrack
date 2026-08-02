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

## 011 — Dos secretos distintos para acceso y refresco

**Fecha:** 2026-07-31 · **Fase:** 3

- **Contexto:** hay dos tipos de token con vidas muy distintas (15 minutos y 7 días).
- **Alternativas:** un solo secreto, distinguiendo los tipos por un campo dentro del token.
- **Decisión:** un secreto por tipo, y además un campo `tipo` que se verifica.
- **Por qué:** con secretos separados, mandar el token de refresco en el encabezado
  `Authorization` simplemente no valida, aunque alguien se equivoque al programar la validación.
  El campo `tipo` es la segunda barrera y deja el propósito escrito dentro del propio token.
  La configuración además exige que los dos secretos sean distintos.

---

## 012 — El mismo error para "no existe el correo" y "contraseña incorrecta"

**Fecha:** 2026-07-31 · **Fase:** 3

- **Contexto:** el login puede fallar por dos razones distintas.
- **Decisión:** los dos casos devuelven exactamente el mismo `401 CREDENCIALES_INVALIDAS`, con el
  mismo mensaje. Y cuando el correo no existe, igual se ejecuta una comparación bcrypt contra un
  hash de relleno.
- **Por qué:** si los errores fueran distintos, cualquiera podría averiguar qué correos tienen
  cuenta probando el login. La comparación de relleno evita que la diferencia de tiempo de
  respuesta (bcrypt tarda ~60 ms) delate lo mismo que el mensaje ya no delata.

---

## 013 — No se comprueba si el correo existe antes de insertar

**Fecha:** 2026-07-31 · **Fase:** 3

- **Contexto:** el registro tiene que rechazar correos repetidos.
- **Alternativas:** consultar primero si existe y después insertar.
- **Decisión:** insertar directo y traducir el error de llave duplicada del índice único.
- **Por qué:** comprobar y luego insertar deja una ventana en la que dos registros simultáneos
  con el mismo correo pasan los dos la comprobación. El índice único no tiene esa ventana. El
  repositorio traduce el error del driver a `ErrDuplicado` y el servicio lo convierte en
  `409 EMAIL_YA_REGISTRADO`.

---

## 014 — Límite de peticiones en memoria, solo en `/auth`

**Fecha:** 2026-07-31 · **Fase:** 3

- **Contexto:** hay que frenar los intentos de fuerza bruta contra el login.
- **Alternativas:** un limitador distribuido con Redis; limitar toda la API.
- **Decisión:** contador de ventana fija en memoria, 20 peticiones por minuto y por IP, aplicado
  solo al grupo `/auth`.
- **Por qué:** es el algoritmo más simple que resuelve el problema real y son 60 líneas que se
  explican completas. Al ser en memoria, el límite es por instancia; con varias réplicas habría
  que mover el contador a Redis, pero eso no aplica al alcance de este proyecto. Limitar el resto
  de la API estorbaría al uso normal (el dashboard hace varias peticiones a la vez).

---

## 015 — Borrar cuentas y categorías: 409 en vez de borrado en cascada

**Fecha:** 2026-08-01 · **Fase:** 4

- **Contexto:** una cuenta o categoría puede tener transacciones que la referencian.
- **Alternativas:** borrado en cascada; dejar los movimientos huérfanos; borrado lógico siempre.
- **Decisión:** `DELETE` borra solo si no hay movimientos; si los hay responde `409` y el cliente
  debe archivar (`archivada: true`).
- **Por qué:** el borrado en cascada puede tirar meses de historial por un clic y no tiene vuelta
  atrás. El campo `archivada` ya existía en el modelo justo para este caso. El conteo previo
  también filtra por `usuario_id`, así que los movimientos de otro usuario no pueden impedirte
  borrar tu propia cuenta.

---

## 016 — El tipo de una categoría con movimientos no se puede cambiar

**Fecha:** 2026-08-01 · **Fase:** 4

- **Contexto:** `tipo` está duplicado en `transacciones` y en `categorias`.
- **Decisión:** al crear o editar una transacción se exige que su `tipo` coincida con el de su
  categoría (`400 TIPO_NO_COINCIDE`), y no se permite cambiar el tipo de una categoría que ya
  tiene movimientos (`409`).
- **Por qué:** la duplicación es deliberada (deja que el reporte de gastos por categoría filtre
  sin resolver la categoría de cada documento), pero solo sirve si se mantiene coherente. Estas
  dos son las únicas puertas por donde puede entrar una incoherencia.

---

## 017 — Las fechas se recortan a milisegundos antes de guardarlas

**Fecha:** 2026-08-01 · **Fase:** 4

- **Contexto:** una prueba de integración falló comparando el `creado_en` que devolvió el POST
  con el que devolvía un GET posterior.
- **Causa:** MongoDB guarda las fechas con precisión de milisegundos y `time.Now()` de Go trae
  nanosegundos. La respuesta del POST traía una hora más precisa que la realmente almacenada.
- **Decisión:** `servicios/tiempo.go` recorta a milisegundos toda fecha antes de guardarla.
- **Por qué:** lo que responde la API tiene que ser exactamente lo que quedó en la base. Si no,
  el frontend compara dos valores del mismo campo y no coinciden.

---

## 018 — El texto de búsqueda se escapa antes de armar la expresión regular

**Fecha:** 2026-08-01 · **Fase:** 4

- **Contexto:** el filtro `busqueda` de transacciones se traduce a un `$regex` de MongoDB.
- **Decisión:** el texto pasa por `regexp.QuoteMeta` antes de entrar en la consulta.
- **Por qué:** sin eso, buscar `.*` recorrería la colección entera (una forma barata de tumbar el
  servidor) y una expresión mal formada haría fallar la consulta con un error interno. Hay una
  prueba de integración que comprueba que buscar `.*` devuelve cero resultados.

---

## 019 — La fecha de un movimiento es un día, anclado a las 12:00 UTC

**Fecha:** 2026-08-02 · **Fase:** 5 · *(resuelve el pendiente "Fase 5 — Zona horaria")*

- **Contexto:** `fecha` llegaba del cliente como un instante y se guardaba tal cual en UTC. Un gasto
  de las 19:00 del 31 de julio en Ciudad de México (UTC−6) es el 1 de agosto en UTC, así que caía
  en el presupuesto y en el reporte del mes equivocado.
- **Alternativas:**
  1. Fijar todo el backend al huso `America/Mexico_City`. Se descartó: ata la aplicación a un país
     y rompe en cuanto alguien viaja o el servidor corre en otra región.
  2. Guardar además el huso del cliente en cada transacción. Se descartó: complica todas las
     agregaciones para un dato que nadie va a consultar.
  3. Tratar la fecha como lo que realmente es: un **día del calendario**, no un instante.
- **Decisión:** `servicios.diaCalendario` toma el año, el mes y el día **en el huso con el que llegó
  la fecha** (`time.Time.Date()` los devuelve así) y los vuelve a construir como las **12:00 UTC**
  de ese mismo día. Los rangos de mes (`modelos.Periodo.Rango`) y los `$year`/`$month` de las
  agregaciones trabajan todos en UTC.
- **Por qué:** nadie apunta un gasto "a las 19:03:22"; lo apunta el día 31. El ancla a mediodía deja
  doce horas de margen a cada lado, así que el día se lee igual desde cualquier huso entre UTC−11 y
  UTC+11. La semilla ya usaba este mismo criterio.
- **Comprobado en:** `servicios/tiempo_test.go` y una petición real con
  `fecha: "2026-07-31T19:00:00-06:00"`, que queda guardada como `2026-07-31T12:00:00Z` y cuenta
  contra el presupuesto de julio.

---

## 020 — El saldo de una cuenta se calcula, no se guarda

**Fecha:** 2026-08-02 · **Fase:** 5 · *(resuelve el pendiente "Fase 5 — Saldo de cuentas")*

- **Contexto:** `cuentas` guarda `saldo_inicial` pero no un saldo actual.
- **Decisión:** `GET /reportes/saldos` calcula `saldo_inicial + ingresos − gastos` con una
  agregación que cruza `cuentas` con `transacciones`.
- **Por qué:** un saldo guardado es una segunda fuente de verdad. Se desincroniza en cuanto una
  transacción se edita o se borra y algo falla a mitad, y a partir de ahí la aplicación miente sin
  que nadie pueda notarlo. Calcularlo cuesta una agregación sobre un índice y siempre es correcto.
- **Contrapartida:** con muchísimos movimientos habría que guardar cortes mensuales. Para el
  volumen de una app de finanzas personales no aplica.

---

## 021 — Una categoría con presupuestos tampoco se puede borrar

**Fecha:** 2026-08-02 · **Fase:** 5

- **Contexto:** la regla de la fase 4 solo miraba las transacciones.
- **Decisión:** `DELETE /categorias/:id` responde `409 CATEGORIA_CON_PRESUPUESTOS` si la categoría
  tiene presupuestos, igual que ya hacía con los movimientos.
- **Por qué:** la consulta de estado de presupuestos hace `$lookup` a `categorias` seguido de
  `$unwind`, y `$unwind` **descarta** los documentos cuyo arreglo quedó vacío. Un presupuesto
  huérfano no daría error: simplemente desaparecería del tablero sin avisar. Es la clase de fallo
  peor, el que no se nota.

---

## 022 — Un `mes` o un `anio` mal escrito en un reporte se rechaza

**Fecha:** 2026-08-02 · **Fase:** 5

- **Contexto:** los filtros del listado ajustan los valores fuera de rango en vez de rechazarlos
  (página 0 pasa a 1). La primera versión de `periodoDeLaConsulta` reutilizó ese criterio y una
  prueba lo cazó: `mes=abc` se colaba como el mes actual.
- **Decisión:** en `/reportes` y `/presupuestos`, si `mes` o `anio` vienen y no son un número
  válido, se responde `400 PERIODO_INVALIDO`. Si no vienen, se usa el mes en curso.
- **Por qué:** un listado con un filtro raro devuelve menos filas y se nota. Un reporte del mes 13
  devolvería ceros, y "no gastaste nada" es una respuesta creíble y falsa. Cuando el error es
  invisible, hay que rechazarlo.

---

## 023 — La importación de CSV es todo o nada

**Fecha:** 2026-08-02 · **Fase:** 6

- **Contexto:** `POST /transacciones/importar` recibe un archivo con muchas filas y alguna puede
  venir mal.
- **Alternativas:**
  1. Insertar las buenas y reportar las malas. Se descartó: si el usuario corrige el archivo y lo
     vuelve a subir, las buenas entran **otra vez**. En una app de dinero eso se paga contando dos
     veces el mismo gasto.
  2. Usar una transacción de MongoDB. No se puede: el servidor está en modo standalone
     ([decisión 003](#003--mongodb-standalone-sin-replica-set)).
- **Decisión:** se valida el archivo **completo** antes de escribir nada. Si una sola fila falla, se
  responde `400 CSV_INVALIDO` con la lista de errores (fila por fila, con el número que se ve en la
  hoja de cálculo) y no se guarda ninguna.
- **Por qué:** sin transacciones multidocumento, validar antes de escribir es la única forma de que
  el resultado sea "todo" o "nada" y nunca "la mitad". Y reintentar es seguro por construcción.
- **Contrapartida:** un archivo de 500 filas con una errata no importa nada. Es molesto, pero es
  predecible; el error dice exactamente qué fila arreglar.

---

## 024 — El CSV lleva nombres, no identificadores, y una marca BOM

**Fecha:** 2026-08-02 · **Fase:** 6

- **Contexto:** el archivo tiene que poder abrirse en Excel y editarse a mano.
- **Decisiones:**
  1. Las columnas `cuenta` y `categoria` van por **nombre**. Un `ObjectID` de 24 caracteres no le
     dice nada a nadie en una hoja de cálculo, y la importación resuelve por nombre (sin distinguir
     mayúsculas ni espacios de sobra), así que lo que exporta la API se puede volver a importar tal
     cual.
  2. El archivo empieza con la **marca BOM** de UTF-8. Excel supone que un `.csv` viene en la
     codificación del sistema y sin ella parte los acentos (*Educación* se ve como *EducaciÃ³n*).
     La importación la quita al leer.
- **Efecto secundario:** si dos categorías se llaman igual salvo por las mayúsculas, una fila no
  puede saber a cuál se refiere. En vez de elegir una en silencio, se responde un error que lo dice.
- **Comprobado en:** una prueba de integración exporta con un usuario y vuelve a importar el archivo
  con otro, sin editarlo, y compara las dos exportaciones.

---

## 025 — Swagger se genera de las anotaciones, y lo generado se versiona

**Fecha:** 2026-08-02 · **Fase:** 6

- **Contexto:** `swaggo` lee comentarios `@Summary`, `@Param`, `@Success`… sobre cada handler y
  genera `backend/docs/` (`docs.go`, `swagger.json`, `swagger.yaml`).
- **Decisión:** las anotaciones van **sobre cada handler**, no en un archivo aparte de stubs, y el
  directorio `backend/docs/` **se versiona** aunque sea código generado.
- **Por qué:**
  - Junto al handler, la anotación se actualiza en el mismo cambio que el código. En un archivo
    aparte se desincroniza en la primera semana.
  - `internal/rutas` importa `docs` por su `init()`, así que **sin ese directorio el proyecto no
    compila**. Ignorarlo obligaría a instalar `swag` antes de poder construir, en la máquina de
    cualquiera y en el CI.
- **Se regenera con** `make swagger` (instala `swag` si falta).
- **Nota de despliegue:** `/swagger` se sirve sin autenticación porque describe la API, no expone
  datos. Aun así, en un despliegue público conviene dejarlo solo en entornos internos.

---

## 026 — Un error al validar el periodo de un reporte se rechaza; los tokens de refresco no se renuevan

**Fecha:** 2026-08-02 · **Fase:** 6

- **Contexto:** al escribir la colección de Postman, una aserción falló: `POST /auth/refresh` no
  devolvía `token_refresco`.
- **Hallazgo:** no era un fallo de la API, sino de la documentación. `/auth/refresh` devuelve
  `RespuestaRefresco` —solo un token de acceso nuevo— porque el de refresco sigue siendo válido
  hasta que expire a los 7 días. La anotación de Swagger decía `RespuestaSesion`.
- **Decisión:** se corrigió la anotación y se dejó la aserción al revés: la prueba de Postman ahora
  comprueba que `token_refresco` **no** viene en la respuesta.
- **Por qué queda anotado:** es el ejemplo de para qué sirve la colección. Las pruebas de Go
  comprueban el comportamiento; la colección comprobó que lo que la API *dice* de sí misma coincide
  con lo que *hace*.

---

## 027 — El 401 se reintenta una vez por petición, no una vez por sesión

**Fecha:** 2026-08-02 · **Fase:** 7

- **Contexto:** el token de acceso vive 15 minutos. El cliente tiene que renovarlo sin que el
  usuario note nada.
- **Alternativas:** (a) decodificar el JWT en el navegador y renovar antes de que expire;
  (b) dejar que la petición falle con `401` y renovar ahí.
- **Decisión:** (b), con tres detalles en `frontend/src/api/cliente.js`:
  1. La marca de reintento va en la config de la petición (`peticion.reintentada`), no en una
     variable del módulo: el límite es *una vez por petición*, no *una vez por sesión*.
  2. Si varias peticiones fallan a la vez, todas se cuelgan de la **misma** promesa de refresco.
  3. `/auth/login`, `/auth/registro` y `/auth/refresh` quedan fuera, listadas una por una y no por
     prefijo, porque `/auth/perfil` empieza igual y esa sí debe reintentarse.
- **Por qué:** (a) obliga a confiar en el reloj del navegador; quien decide si un token sirve es el
  servidor. Sin el punto 1, un token de refresco vencido daría `401 → refresh → 401 → refresh` en
  bucle. Sin el punto 2, el segundo refresco llegaría con el token viejo y cerraría la sesión sin
  motivo. Sin el punto 3, un error de credenciales se convertiría en un cierre de sesión.

---

## 028 — Las gráficas no usan el verde y el rojo de la aplicación

**Fecha:** 2026-08-02 · **Fase:** 7

- **Contexto:** ingresos en verde y gastos en rojo es la convención de la app y se usa en cifras,
  etiquetas y montos. Lo natural era llevarla también a la gráfica de barras.
- **Hallazgo:** medido con un validador de paletas —no a ojo—, `#16A34A` contra `#DC2626` se
  separan **ΔE 5.0** en simulación de deuteranopia (y **1.1** con las variantes del tema oscuro).
  Para una de cada doce personas aproximadamente, esas dos barras son el mismo color.
- **Decisión:** las series de las gráficas usan `--serie-ingresos: #059669` (esmeralda) y
  `--serie-gastos: #EA580C` (naranja): **ΔE 10.1**, y pasan las cinco comprobaciones —banda de
  luminosidad, croma mínimo, separación bajo daltonismo, separación con visión normal y contraste—
  contra las **dos** superficies, la clara y la oscura. Son el mismo par en los dos temas.
- **Por qué:** en una gráfica el color es lo único que distingue una serie de otra. El verde y el
  rojo se quedan donde el color **acompaña a una palabra** ("Ingreso", "Gasto", "Excedido") y por
  tanto no es la única pista. Además, el rojo ya está reservado para los errores y para el estado
  "excedido": usarlo también como color de serie mezclaría estado con identidad.

---

## 029 — Ninguna gráfica es la única forma de leer un dato

**Fecha:** 2026-08-02 · **Fase:** 7

- **Contexto:** los colores de las categorías **los elige el usuario**. La aplicación no puede
  validarlos de antemano ni garantizar que dos categorías vecinas se distingan.
- **Decisión:** cada gráfica lleva debajo un `<details>` con los mismos números en una tabla, una
  leyenda con el nombre de cada serie, y un globo que también nombra lo que muestra. El pastel
  agrupa a partir de la sexta categoría en una porción "Otras (n)"; la cola sigue entera en la
  tabla.
- **Por qué:** si la única forma de leer un valor es pasar el ratón por encima, el dato no existe
  para quien navega con teclado, usa un lector de pantalla o imprime la página. Y como los colores
  son datos del usuario, la interfaz tiene que seguir funcionando con cualquier combinación.

---

## 030 — Al recargar, lo viejo se atenúa; el esqueleto sale una sola vez

**Fecha:** 2026-08-02 · **Fase:** 7

- **Contexto:** el tablero, los reportes y el listado se vuelven a pedir cada vez que cambia el mes,
  la página o un filtro.
- **Decisión:** el esqueleto se pinta solo cuando todavía no hay datos. En las recargas posteriores
  se conserva lo que ya estaba a media opacidad (`[data-cargando]`) hasta que llegan los nuevos.
- **Por qué:** volver al esqueleto en cada recarga hace saltar la maquetación y pierde el sitio
  donde estaba el usuario. `usePeticion` no borra `datos` al empezar una petición nueva, que es lo
  que hace posible esta transición.

---

## 031 — Los tokens viven en `localStorage`

**Fecha:** 2026-08-02 · **Fase:** 7

- **Contexto:** hay que guardar el token de acceso y el de refresco entre recargas.
- **Alternativas:** `localStorage`, o una cookie `httpOnly` emitida por el backend.
- **Decisión:** `localStorage`, con todo el acceso encapsulado en `api/sesion.js` y envuelto en
  `try` (en modo incógnito de Safari, escribir lanza).
- **Por qué:** la cookie `httpOnly` es más segura contra XSS, pero exigiría que el backend la
  emitiera, la validara y gestionara CSRF, y eso cambia el contrato de la fase 3, que es sin estado
  y espera el token en `Authorization`. **Queda como limitación conocida**, no como decisión
  cerrada: si en la fase 10 se decide endurecerlo, el único archivo que cambia es `sesion.js`.

---

## 032 — `Content-Disposition` se expone en CORS

**Fecha:** 2026-08-02 · **Fase:** 7

- **Contexto:** al exportar el CSV, el frontend quiere guardar el archivo con el nombre que propone
  el servidor.
- **Hallazgo:** el navegador solo deja leer un puñado de encabezados de una respuesta entre
  orígenes, y `Content-Disposition` no está entre ellos. En desarrollo no se notaba porque el proxy
  de Vite hace que todo sea el mismo origen.
- **Decisión:** añadir `Content-Disposition` a `Access-Control-Expose-Headers` en el middleware de
  CORS, con su prueba. El cliente igualmente arma un nombre de respaldo con la fecha de hoy si el
  encabezado no llega.
- **Por qué:** es un cambio de una línea en el backend que evita un fallo silencioso el día que la
  API y el frontend vivan en dominios distintos.

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
- ~~**Fase 5 — Zona horaria**~~ → resuelto en la [decisión 019](#019--la-fecha-de-un-movimiento-es-un-día-anclado-a-las-1200-utc).
- ~~**Fase 5 — Saldo de cuentas**~~ → resuelto en la [decisión 020](#020--el-saldo-de-una-cuenta-se-calcula-no-se-guarda).
- **Fase 7 — Alertas solo al crear:** sigue abierto. `POST /transacciones` avisa si el gasto rebasa
  el presupuesto, pero `PUT` no, y el frontend solo enseña la alerta al crear. Editar el monto de un
  gasto también puede cruzar el umbral. La salida más limpia es extender el campo `alerta` al `PUT`,
  no que el cliente haga una segunda consulta después de cada guardado.
- **Fase 7 — Importación sin vista previa:** sigue abierto. El CSV se valida entero y se guarda o se
  rechaza en la misma petición; el frontend enseña las filas a corregir, pero no lo que se va a
  importar. Una vista previa necesitaría un endpoint que valide sin escribir.
- **Fase 7 — Frontend sin pruebas automatizadas:** la verificación de la fase 7 fue manual, contra
  la API viva y con un recorrido guiado en un navegador sin ventana. Falta decidir si entran Vitest
  y Testing Library, que serían las primeras dependencias de prueba del frontend.
- **Fase 1 — La semilla está anclada a julio de 2026:** `ANIO_FINAL` y `MES_FINAL` en
  `database/02_insertar_datos.js`. Como el tablero abre en el mes en curso, según cuándo se haga la
  demostración lo primero que se ve pueden ser ceros. Se re-ancla cambiando esas dos líneas y
  volviendo a correr `make seed`.
