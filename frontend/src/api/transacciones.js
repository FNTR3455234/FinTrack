import { cliente } from './cliente.js'

// Listar es la unica llamada del proyecto que devuelve tambien "meta": la
// paginacion vive ahi, y la tabla la necesita para pintar "pagina 2 de 6".
export function listar(params, opciones) {
  return cliente.get('/transacciones', { params, ...opciones }).then((r) => ({
    datos: r.data.datos,
    meta: r.data.meta,
  }))
}

export function obtener(id) {
  return cliente.get(`/transacciones/${id}`).then((r) => r.data.datos)
}

// Crear devuelve la transaccion y, si el gasto dejo su categoria cerca del
// limite o por encima, tambien el campo "alerta". Se devuelve tal cual para que
// la pantalla decida si la muestra.
export function crear(datos) {
  return cliente.post('/transacciones', datos).then((r) => r.data.datos)
}

export function actualizar(id, datos) {
  return cliente.put(`/transacciones/${id}`, datos).then((r) => r.data.datos)
}

export function eliminar(id) {
  return cliente.delete(`/transacciones/${id}`).then(() => true)
}

// exportarCSV pide el archivo con los mismos filtros del listado.
//
// responseType blob y no text: el archivo lleva marca BOM al principio para que
// Excel lo abra en UTF-8, y tratarlo como texto haria que axios intentara
// interpretarlo y se perderia.
export function exportarCSV(params) {
  return cliente
    .get('/transacciones/exportar', { params, responseType: 'blob' })
    .then((r) => ({ blob: r.data, nombre: nombreDelArchivo(r.headers) }))
}

// importarCSV sube el archivo. No se fija el Content-Type a mano: el navegador
// tiene que ponerlo con el "boundary" del multipart, y escribirlo nosotros lo
// romperia.
export function importarCSV(archivo) {
  const cuerpo = new FormData()
  cuerpo.append('archivo', archivo)
  return cliente.post('/transacciones/importar', cuerpo).then((r) => r.data.datos)
}

// nombreDelArchivo saca el nombre que propuso el servidor en Content-Disposition.
//
// Si no llega (por ejemplo si algun dia la API vive en otro origen y no expone
// ese encabezado), se arma uno con la fecha de hoy: perder el nombre no debe
// impedir la descarga.
function nombreDelArchivo(cabeceras) {
  const disposicion = cabeceras?.['content-disposition'] || ''
  const encontrado = /filename="?([^";]+)"?/i.exec(disposicion)
  if (encontrado) return encontrado[1]
  return `transacciones-${new Date().toISOString().slice(0, 10)}.csv`
}
