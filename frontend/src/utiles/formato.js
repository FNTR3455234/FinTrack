// Formato de dinero, fechas y porcentajes. Todo en un sitio para que las cifras
// se vean iguales en la tabla, en el tablero y en las graficas.

const MESES = [
  'enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio',
  'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre',
]

// Construir un Intl.NumberFormat es caro y aqui se llama en cada celda de la
// tabla, asi que se guardan los ya creados por moneda.
const formateadores = new Map()

function formateador(moneda) {
  if (!formateadores.has(moneda)) {
    formateadores.set(
      moneda,
      new Intl.NumberFormat('es-MX', {
        style: 'currency',
        currency: moneda,
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }),
    )
  }
  return formateadores.get(moneda)
}

// dinero formatea un monto. Si la moneda del usuario no es un codigo ISO valido
// Intl lanza, asi que se cae a MXN en vez de romper la pantalla entera.
export function dinero(monto, moneda = 'MXN') {
  const numero = Number(monto) || 0
  try {
    return formateador(moneda).format(numero)
  } catch {
    return formateador('MXN').format(numero)
  }
}

// dineroCorto se usa en los ejes de las graficas, donde "$12,450.00" no cabe.
export function dineroCorto(monto) {
  const numero = Number(monto) || 0
  if (Math.abs(numero) >= 1000) return `$${Math.round(numero / 1000)}k`
  return `$${Math.round(numero)}`
}

// fechaCorta convierte "2026-07-31T12:00:00Z" en "31 jul 2026".
//
// El dia se toma de la propia cadena y no de new Date(...).getDate(): la API
// ancla las fechas a las 12:00 UTC (ver decision 019) y leerlas en la zona
// horaria del navegador podria correr el dia. Cortando la cadena, lo que se ve
// es exactamente el dia que esta guardado y el mismo que usan los filtros.
export function fechaCorta(iso) {
  const dia = soloDia(iso)
  if (!dia) return ''
  const [anio, mes, numero] = dia.split('-')
  return `${Number(numero)} ${MESES[Number(mes) - 1].slice(0, 3)} ${anio}`
}

// soloDia devuelve la parte AAAA-MM-DD, que es lo que espera <input type="date">.
export function soloDia(iso) {
  if (!iso) return ''
  return String(iso).slice(0, 10)
}

// aFechaAPI convierte el "2026-07-31" de un <input type="date"> en el instante
// que espera la API. Las 12:00 UTC son las mismas que usa el backend para
// anclar el dia calendario: asi lo que se manda y lo que se guarda coinciden.
export function aFechaAPI(dia) {
  return `${dia}T12:00:00Z`
}

// hoy devuelve la fecha de hoy en AAAA-MM-DD segun el calendario del usuario,
// que es el que ve en su pantalla al abrir el formulario.
export function hoy() {
  const ahora = new Date()
  const mes = String(ahora.getMonth() + 1).padStart(2, '0')
  const dia = String(ahora.getDate()).padStart(2, '0')
  return `${ahora.getFullYear()}-${mes}-${dia}`
}

// nombreMes devuelve "julio" a partir del 7. Los periodos de la API son mes y
// anio sueltos, no fechas.
export function nombreMes(mes) {
  return MESES[mes - 1] || ''
}

// periodoLargo arma "julio de 2026" para los titulos.
export function periodoLargo({ mes, anio }) {
  return `${nombreMes(mes)} de ${anio}`
}

// porcentaje redondea a un decimal y quita el ".0" cuando sobra.
export function porcentaje(valor) {
  const numero = Number(valor) || 0
  return `${numero % 1 === 0 ? numero : numero.toFixed(1)}%`
}

export { MESES }
