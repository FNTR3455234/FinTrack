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

_(Pendiente en la fase 4: mostrarlo con el filtro de un repositorio real.)_

## Modelo de datos

Ver [`database/modelo.md`](../database/modelo.md) para el diagrama de colecciones y relaciones.

## Autenticación

Dos tokens firmados con secretos distintos: el de **acceso** (15 min) acompaña cada petición y el
de **refresco** (7 días) solo sirve para pedir uno de acceso nuevo. El detalle está en
[`../backend/README.md`](../backend/README.md#autenticación).

_(Pendiente en la fase 7: el reintento del interceptor de axios ante un 401.)_

## Decisiones

Ver [`decisiones.md`](decisiones.md).
