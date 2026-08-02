# Arquitectura de FinTrack

> Documento en construcción. Se completa en la **fase 8**, cuando ya existan todas las piezas.
> Aquí queda el esqueleto para ir llenándolo conforme avanzan las fases.

## Vista general

_(Pendiente: diagrama de los tres contenedores — MongoDB, API en Go, frontend servido por nginx —
y cómo se hablan entre sí.)_

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

_(Pendiente en la fase 7: el reintento del interceptor de axios ante un 401.)_

## Decisiones

Ver [`decisiones.md`](decisiones.md).
