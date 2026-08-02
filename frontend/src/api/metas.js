import { cliente } from './cliente.js'

// Las metas de ahorro y sus aportaciones.
//
// No se usa el crudDe de catalogos.js: el listado devuelve un objeto
// { metas, resumen } en vez de un arreglo, y ademas hay dos rutas mas para el
// sub-recurso de aportaciones. Forzarlo dentro del molde comun costaria mas
// explicaciones de las que ahorra.

export function listar(params, opciones) {
  return cliente.get('/metas', { params, ...opciones }).then((r) => r.data.datos)
}

// Devuelve la meta con el detalle de cada aportacion, que es lo que explica de
// donde sale el total.
export function obtener(id, opciones) {
  return cliente.get(`/metas/${id}`, opciones).then((r) => r.data.datos)
}

export function crear(datos) {
  return cliente.post('/metas', datos).then((r) => r.data.datos)
}

export function actualizar(id, datos) {
  return cliente.put(`/metas/${id}`, datos).then((r) => r.data.datos)
}

export function eliminar(id) {
  return cliente.delete(`/metas/${id}`).then(() => true)
}

// Las aportaciones cuelgan de su meta: no existen por su cuenta y por eso su
// ruta lleva dentro el identificador de la meta.
export function aportar(metaID, datos) {
  return cliente.post(`/metas/${metaID}/aportaciones`, datos).then((r) => r.data.datos)
}

export function quitarAportacion(metaID, aportacionID) {
  return cliente.delete(`/metas/${metaID}/aportaciones/${aportacionID}`).then(() => true)
}
