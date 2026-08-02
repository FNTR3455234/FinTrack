# Arquitectura de FinTrack

## Vista general

Tres contenedores en una red privada de Docker. **Solo uno publica un puerto.**

```mermaid
flowchart LR
    N["🌐 Navegador"]

    subgraph red["red interna de Docker (fintrack-prod_interna)"]
        direction LR
        W["<b>web</b><br/>nginx 1.27-alpine<br/>bundle de React<br/>:8080"]
        A["<b>api</b><br/>Go 1.25 · Gin<br/>binario estático<br/>:8080"]
        M[("<b>mongo</b><br/>MongoDB 7<br/>:27017")]
    end

    V[["volumen<br/>mongo_datos"]]

    N -->|"http://localhost:8080"| W
    W -->|"/ · /assets/*<br/>archivos estáticos"| W
    W -->|"/api/* · /swagger<br/>proxy_pass"| A
    A -->|"driver oficial<br/>mongo-driver/v2"| M
    M --- V
```

Lo que hay detrás de ese dibujo:

- **Un solo puerto publicado.** `mongo` y `api` no exponen nada a la máquina anfitriona. Para
  llegar a ellos hay que estar dentro de la red o entrar con `docker compose exec`. Una base de
  datos con el 27017 abierto al mundo es de las formas más comunes de filtrar datos.
- **Un solo origen para el navegador.** nginx sirve el frontend y además pasa `/api` a la API, así
  que el navegador nunca ve dos dominios: no hay CORS de por medio y `VITE_API_BASE=/api/v1` vale
  igual en desarrollo (proxy de Vite) que en producción (proxy de nginx).
- **El estado vive en un solo sitio.** `api` y `web` son desechables: se pueden borrar y volver a
  crear sin perder nada. Todo lo que persiste está en el volumen `mongo_datos`.
- **Arranque por orden de salud.** `api` no arranca hasta que Mongo responde al `ping`, y `web` no
  arranca hasta que `/api/v1/health` responde. Sin eso la API arrancaría primero, no conectaría y
  se apagaría (`main.go` hace `os.Exit(1)` si la conexión falla).

### Las dos imágenes

Las dos son multi-etapa: una etapa compila y la otra solo ejecuta. Lo que se manda a un servidor
es superficie de ataque, y ni el compilador ni el código fuente hacen falta para ejecutar.

| Imagen | Compila con | Ejecuta sobre | Tamaño | Usuario |
|---|---|---|---|---|
| `fintrack-api` | `golang:1.25-alpine` | `alpine:3.21` | ~64 MB | `fintrack` (uid 10001) |
| `fintrack-web` | `node:22-alpine` | `nginx:1.27-alpine` | ~83 MB | `nginx` |

El binario de Go se compila con `CGO_ENABLED=0`, así que es estático y no depende de la libc del
sistema; de Alpine solo se usan los certificados y `wget` para el *healthcheck*. La versión se
graba al compilar con `-ldflags`, así que `/api/v1/health` dice qué versión está desplegada.

### Los dos stacks de Compose

| Archivo | Proyecto | Qué levanta | Para qué |
|---|---|---|---|
| `docker-compose.dev.yml` | `fintrack-dev` | Solo MongoDB | Desarrollo: la API y el frontend corren a mano, con recarga en caliente |
| `docker-compose.yml` | `fintrack-prod` | Los tres servicios | La entrega: `docker compose up` y listo |

Los nombres de proyecto son **explícitos y distintos** a propósito. Sin ellos Compose usa el nombre
de la carpeta, que es el mismo para los dos archivos, y los dos stacks acaban compartiendo el
volumen `mongo_datos`. Como la imagen de Mongo solo crea el usuario administrador la primera vez,
con el directorio de datos vacío, el segundo stack se encuentra un volumen que ya trae el usuario
del primero y **no puede autenticarse nunca**. Ver la [decisión 033](decisiones.md).

## Capas del backend

El backend sigue un flujo en una sola dirección; cada capa solo conoce a la siguiente:

```
handlers  ->  servicios  ->  repositorios  ->  MongoDB
(HTTP)        (reglas)       (queries)
```

- **handlers** — parsean la petición, llaman al servicio y responden con el formato uniforme.
  No contienen reglas de negocio ni consultas.
- **servicios** — reglas de negocio y validaciones que cruzan entidades. Reciben los repositorios
  como interfaz, lo que permite mockearlos en las pruebas unitarias.
- **repositorios** — única capa que habla con MongoDB.
- **middleware, errores, modelos, config, db, rutas** — piezas transversales.

_(Pendiente: detallar el contrato de cada capa con ejemplos reales de código.)_

## Capas del frontend

El mismo principio, en el otro lado del cable:

```
paginas  ->  hooks  ->  api  ->  axios  ->  HTTP
(vista)      (estado)   (endpoints)
```

- **paginas** — la vista y el estado de la pantalla. No arman URLs ni tocan `localStorage`.
- **componentes** — lo que se repite (`Boton`, `Campo`, `Modal`, `Tarjeta`, las gráficas). Nunca
  importan de una página.
- **hooks** — `usePeticion` (leer: datos, cargando, error, recargar, con cancelación),
  `useAccion` (escribir: ocupado y error), `usePeriodo`, `useRetardo`.
- **api** — una función por endpoint. Devuelven ya el contenido de `datos`: el sobre
  `{ datos, meta }` es un detalle del transporte y no llega a los componentes.
- **contexto** — `AuthContexto` es el **único** sitio que escribe o borra tokens; `TemaContexto`
  es el único que toca el atributo `data-tema` de `<html>`.

El interceptor de `api/cliente.js` es la pieza con más reglas del cliente: adjunta el token,
renueva **una sola vez por petición** ante un `401` y comparte una única promesa de refresco entre
las peticiones que fallen a la vez (ver [decisión 027](decisiones.md)).

## Aislamiento por usuario

Regla innegociable: ninguna consulta llega a MongoDB sin filtrar por `usuario_id`, y ese
`usuario_id` sale siempre del token JWT, nunca del cuerpo de la petición.

El recorrido completo:

1. `middleware.Autenticacion` lee el encabezado `Authorization: Bearer <token>`, valida la firma
   y la vigencia, y saca el id del usuario del campo `sub` del token.
2. Lo guarda en el contexto de Gin con la llave `usuario_id`.
3. Los handlers lo leen con `middleware.UsuarioID(c)` y se lo pasan al servicio como argumento.
4. El repositorio lo mete en el filtro de **toda** consulta a MongoDB.

Si el id viniera del cuerpo o de la query, cualquiera podría pedir los datos de otro cambiando un
valor. Por eso los DTO de entrada **no tienen** un campo `usuario_id`: no hay forma de mandarlo.

En el repositorio, el paso 4 son dos funciones en `repositorios/comunes.go` por las que pasa toda
operación de un solo documento:

```go
func deUsuario(usuarioID bson.ObjectID) bson.M {
	return bson.M{"usuario_id": usuarioID}
}

func suyoPorID(usuarioID, id bson.ObjectID) bson.M {
	return bson.M{"_id": id, "usuario_id": usuarioID}
}
```

Por eso editar o borrar algo ajeno responde **404 y no 403**: un 403 confirmaría que el recurso
existe. Para el intruso, sencillamente no está.

### Los dos puntos donde el cliente elige un identificador

1. **Al crear una transacción** llegan `cuenta_id` y `categoria_id` en el cuerpo.
   `servicios.validarReferencias` comprueba las dos filtrando por usuario, así que un id ajeno
   "no existe". Lo mismo hace `Presupuestos.validarCategoria`.
2. **En las agregaciones de reportes** (fase 5), donde el riesgo es peor porque cruzan colecciones:
   un `$lookup` al que se le olvide el `usuario_id` sumaría dinero ajeno sin que ningún 403 lo
   delate. Por eso el `$lookup` con `let` + `pipeline` de
   `reportes_presupuestos.go` compara `$$usr` además de la categoría, aunque el `$match` inicial ya
   haya filtrado. Hay una prueba de integración con dos usuarios reales que lo comprueba sobre las
   cinco agregaciones.

## Modelo de datos

Ver [`database/modelo.md`](../database/modelo.md) para el diagrama de colecciones y relaciones.

## Autenticación

Dos tokens firmados con secretos distintos: el de **acceso** (15 min) acompaña cada petición y el
de **refresco** (7 días) solo sirve para pedir uno de acceso nuevo. El detalle está en
[`../backend/README.md`](../backend/README.md#autenticación).

El recorrido completo, incluido lo que pasa cuando el token caduca a mitad de una sesión:

```mermaid
sequenceDiagram
    participant N as Navegador
    participant X as interceptor<br/>(api/cliente.js)
    participant A as API
    participant M as MongoDB

    N->>A: POST /auth/login
    A->>M: buscar por email
    M-->>A: usuario + hash
    A->>A: bcrypt.CompareHashAndPassword
    A-->>N: token_acceso (15 min) + token_refresco (7 días)

    Note over N,X: 15 minutos después…

    N->>X: GET /transacciones
    X->>A: + Authorization: Bearer <acceso>
    A-->>X: 401 TOKEN_VENCIDO

    rect rgb(240, 245, 250)
        Note over X: un solo reintento, marcado en la propia petición
        X->>A: POST /auth/refresh { token_refresco }
        A-->>X: token_acceso nuevo
        X->>A: reintenta GET /transacciones
        A->>M: filtro { usuario_id: <del token> }
        M-->>A: solo los suyos
        A-->>N: 200
    end
```

Si el refresco también falla, el interceptor borra la sesión y avisa a `AuthContexto`, que es quien
lleva al usuario al login. El cliente de axios no conoce react-router: solo avisa.

## Integración continua

`.github/workflows/ci.yml`, en cada `push` a `main` y en cada pull request:

| Trabajo | Qué comprueba |
|---|---|
| `backend` | `gofmt`, `go vet` y las pruebas **contra un MongoDB de verdad**, con cobertura |
| `frontend` | Que el bundle compila, y publica su tamaño en el resumen |
| `imagenes` | Que los dos `Dockerfile` construyen |
| `contrato` | Levanta el **stack completo** con Compose y le pasa la colección de Postman |

Los tres primeros van en paralelo; `contrato` espera a que pasen, porque tarda minutos y no tiene
sentido gastarlos si las pruebas unitarias ya fallaron. Las pruebas de integración usan un MongoDB
real y no un doble: comprueban agregaciones, índices únicos y el `$jsonSchema`, y eso un doble no
lo puede imitar sin acabar imitando también los errores.

## Decisiones

Ver [`decisiones.md`](decisiones.md).
