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
`usuario_id` sale siempre del token JWT (inyectado en el contexto de Gin por el middleware de
autenticación), nunca del cuerpo de la petición.

_(Pendiente: describir cómo el middleware inyecta el id y cómo lo consume cada repositorio.)_

## Modelo de datos

Ver [`database/modelo.md`](../database/modelo.md) para el diagrama de colecciones y relaciones.

## Autenticación

_(Pendiente: flujo de access token de 15 min + refresh token de 7 días, y el reintento del
interceptor de axios ante un 401.)_

## Decisiones

Ver [`decisiones.md`](decisiones.md).
