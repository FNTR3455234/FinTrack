import { cliente } from './cliente.js'

// Los cinco reportes de solo lectura. Todos aceptan mes y anio; sin ellos, la
// API responde el mes en curso.
function reporte(ruta) {
  return (params, opciones) => cliente.get(ruta, { params, ...opciones }).then((r) => r.data.datos)
}

// Las dos consultas relacionales de la entrega academica.
export const gastosPorCategoria = reporte('/reportes/gastos-por-categoria')
export const estadoPresupuestos = reporte('/reportes/estado-presupuestos')

export const resumen = reporte('/reportes/resumen')
export const tendencia = reporte('/reportes/tendencia')
export const saldos = reporte('/reportes/saldos')
