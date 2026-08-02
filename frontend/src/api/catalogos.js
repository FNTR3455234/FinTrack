import { cliente } from './cliente.js'

// Cuentas y categorias comparten exactamente la misma forma de CRUD, asi que
// las cinco funciones se generan una sola vez con crudDe. Escribirlas dos veces
// solo daria dos sitios donde equivocarse.
//
// El identificador del usuario no aparece por ningun lado: lo pone el servidor
// a partir del token. Si estuviera aqui seria un dato que el cliente puede
// elegir, que es justo lo que no debe pasar.
function crudDe(base) {
  return {
    listar: (params, opciones) =>
      cliente.get(base, { params, ...opciones }).then((r) => r.data.datos),
    obtener: (id) => cliente.get(`${base}/${id}`).then((r) => r.data.datos),
    crear: (datos) => cliente.post(base, datos).then((r) => r.data.datos),
    actualizar: (id, datos) => cliente.put(`${base}/${id}`, datos).then((r) => r.data.datos),
    eliminar: (id) => cliente.delete(`${base}/${id}`).then(() => true),
  }
}

export const cuentas = crudDe('/cuentas')
export const categorias = crudDe('/categorias')
export const presupuestos = crudDe('/presupuestos')
