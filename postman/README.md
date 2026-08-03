# Entregable 4 — Pruebas de la API con Postman

Colección completa de la API REST de FinTrack: **53 peticiones** repartidas en 11 carpetas, con
**139 aserciones** que se ejecutan solas.

| Archivo | Qué es |
|---|---|
| [`FinTrack.postman_collection.json`](FinTrack.postman_collection.json) | La colección: todas las peticiones con sus pruebas |
| [`FinTrack.postman_environment.json`](FinTrack.postman_environment.json) | Entorno local: `base_url` y las credenciales del usuario de ejemplo |
| [`ejemplo.csv`](ejemplo.csv) | Archivo de muestra para probar la importación |
| `capturas/` | Capturas de pantalla de las ejecuciones |

## Herramientas empleadas

| Herramienta | Versión | Para qué |
|---|---|---|
| Postman | 11.x | Cliente gráfico: explorar la API y ver las respuestas |
| Newman | 6.x | El mismo ejecutor, pero en línea de comandos (`make postman`) |
| Node.js | 18+ | Solo para Newman; Postman no lo necesita |

## Preparación

La colección habla con una API que tiene que estar viva y con los datos de ejemplo cargados:

```bash
make up            # Windows sin make:  .\make.ps1 up
make seed          # carga el usuario demo con 6 meses de historia
make dev           # levanta la API en http://localhost:8080
```

## Cómo se usa

### Desde Postman

1. **Import** → arrastra los dos archivos `.json`.
2. Arriba a la derecha, elige el entorno **FinTrack local**.
3. Corre primero **Autenticación → Login (usuario demo)**. Guarda el token solo; el resto de las
   peticiones lo heredan de la colección.
4. Ya puedes lanzar cualquier otra.

O directamente **Run collection** para ejecutarla entera de arriba abajo.

### Desde la terminal

```bash
make postman       # Windows:  .\make.ps1 postman
```

Salida real de la última ejecución:

```
┌─────────────────────────┬──────────────────┬──────────────────┐
│                         │         executed │           failed │
├─────────────────────────┼──────────────────┼──────────────────┤
│              iterations │                1 │                0 │
│                requests │               53 │                0 │
│            test-scripts │              107 │                0 │
│              assertions │              139 │                0 │
└─────────────────────────┴──────────────────┴──────────────────┘
```

### ¿Prefieres Bruno?

La misma colección está también en [`bruno/`](../bruno/README.md), como archivos de texto
versionables, uno por petición. **No es una segunda colección que mantener**: la genera
`make bruno` a partir de este `.json`.

## Qué contiene

| Carpeta | Peticiones | Qué comprueba |
|---|---|---|
| **Salud** | 1 | `/health` responde y Mongo contesta |
| **Autenticación** | 5 | Registro, login, refresco y perfil |
| **Cuentas** | 4 | CRUD completo |
| **Categorías** | 4 | CRUD y filtro por tipo |
| **Transacciones** | 6 | CRUD, filtros, búsqueda y paginación |
| **Presupuestos** | 4 | CRUD y periodo |
| **Reportes** | 5 | Las consultas relacionales 1 y 2, resumen, tendencia y saldos |
| **CSV** | 2 | Exportar e importar |
| **Metas de ahorro** | 11 | Consulta relacional 3, aportaciones y el borrado en cascada |
| **Errores** | 8 | Los casos que **tienen** que fallar |
| **Limpieza** | 3 | Borra lo que creó la colección |

### Se puede correr las veces que haga falta

Es la parte que más trabajo dio y la que hace que la colección sirva de verdad:

- **Registro** genera un correo distinto en cada ejecución (`postman_<marca de tiempo>@fintrack.mx`)
  con un script previo, porque el email es único en la base.
- **Limpieza** borra al final la cuenta y la categoría que se crearon al principio.
- Los identificadores viajan solos entre peticiones: cada `POST` guarda el `id` que devolvió en una
  variable de colección (`cuenta_id`, `categoria_id`, `transaccion_id`, `presupuesto_id`) y las
  peticiones siguientes lo usan. No hay que copiar y pegar nada.

### El token se guarda solo

La colección tiene autenticación **bearer a nivel de colección** apuntando a `{{token_acceso}}`, y
las peticiones de login y registro lo escriben desde su script de pruebas:

```javascript
const datos = pm.response.json().datos;
pm.collectionVariables.set("token_acceso", datos.token_acceso);
pm.collectionVariables.set("token_refresco", datos.token_refresco);
```

Las peticiones públicas (`/health`, `/auth/*`) llevan `auth: noauth` para no mandar un token que
no hace falta.

## La carpeta "Errores"

No es relleno: es donde se comprueba que la API falla **como debe**.

| Petición | Se espera | Por qué importa |
|---|---|---|
| Sin token | `401 NO_AUTENTICADO` | |
| Token inventado | `401 TOKEN_INVALIDO` | La firma se verifica de verdad |
| Id de otro usuario | `404` | Y **no 403**: un 403 confirmaría que el recurso existe |
| Id mal escrito | `400 ID_INVALIDO` | |
| Datos inválidos | `400` con **3 o más** detalles | El error enumera todos los campos malos, no solo el primero |
| Periodo imposible (`mes=13`) | `400 PERIODO_INVALIDO` | Devolver ceros se leería como "no gastaste nada" |
| Ruta que no existe | `404 RUTA_NO_ENCONTRADA` | Hasta el 404 usa el formato JSON de la API |
| Método equivocado | `405` | |

## Detalles de las pruebas

Además del código de estado, las aserciones miran el contenido:

- **Gastos por categoría** viene ordenado de mayor a menor y cada fila trae el nombre de su
  categoría (o sea, el `$lookup` funcionó).
- **Estado de presupuestos**: el semáforo solo toma los valores `ok`, `alerta` o `excedido`.
- **Resumen**: `ingresos − gastos` es igual a `balance` con menos de un centavo de diferencia.
- **Tendencia** de 6 meses devuelve exactamente 6 puntos, aunque algún mes esté vacío.
- **Perfil** nunca incluye la palabra `password` en la respuesta.
- **Exportar CSV** baja con `Content-Type: text/csv` y `Content-Disposition: attachment`.
- Y una prueba **a nivel de colección**, que corre en las 53: ninguna respuesta puede tardar más de
  dos segundos.

## Importar un CSV

Postman no guarda archivos dentro de la colección, así que la petición **CSV → Importar desde CSV**
llega con el campo vacío: hay que elegir a mano [`ejemplo.csv`](ejemplo.csv) en el campo `archivo`.

Ese archivo trae dos movimientos válidos contra las cuentas y categorías del usuario de ejemplo.
Para ver el camino del error, cámbiale el nombre de la cuenta por uno que no exista: responde `400`
con la fila exacta y **no guarda ninguna** de las dos.

## Notas

- El entorno guarda `password_demo` como variable de tipo `secret`. Es la contraseña del usuario de
  ejemplo, que ya está en el repositorio: no hay ningún secreto real aquí.
- La misma API está documentada de forma interactiva en
  **[http://localhost:8080/swagger](http://localhost:8080/swagger)** (`make swagger` la regenera).
  Postman prueba la API; Swagger la describe.
