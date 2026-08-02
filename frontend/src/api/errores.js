// Traduce cualquier fallo de axios a una forma unica que el resto de la app
// pueda mostrar sin preguntarse de donde viene.
//
// La API siempre responde los errores igual:
//   { "error": { "codigo": "...", "mensaje": "...", "detalles": [...] } }
// pero un fallo de red, un timeout o el servidor apagado no traen cuerpo. Sin
// esta capa, cada pantalla tendria que mirar error.response?.data?.error?...
// y decidir que hacer si no esta.

// Codigos que inventa el cliente porque no vienen del servidor.
export const CODIGO_SIN_RED = 'SIN_RED'
export const CODIGO_CANCELADO = 'CANCELADO'

export class ErrorAPI extends Error {
  constructor({ codigo, mensaje, detalles = [], estado = 0 }) {
    super(mensaje)
    this.name = 'ErrorAPI'
    this.codigo = codigo
    this.detalles = detalles
    this.estado = estado
  }

  // Texto listo para pintar: el mensaje y, si el servidor detallo campo por
  // campo (los 400 de validacion), tambien esa lista.
  get texto() {
    if (this.detalles.length === 0) return this.message
    return `${this.message} ${this.detalles.join(' ')}`
  }
}

// deAxios recibe lo que sea que haya rechazado la promesa y devuelve un
// ErrorAPI. Nunca lanza: es la ultima linea antes de la interfaz.
export function deAxios(error) {
  if (error instanceof ErrorAPI) return error

  if (error?.code === 'ERR_CANCELED') {
    return new ErrorAPI({
      codigo: CODIGO_CANCELADO,
      mensaje: 'La peticion se cancelo.',
    })
  }

  const cuerpo = error?.response?.data?.error
  if (cuerpo?.codigo) {
    return new ErrorAPI({
      codigo: cuerpo.codigo,
      mensaje: cuerpo.mensaje || 'Ocurrio un error.',
      detalles: cuerpo.detalles || [],
      estado: error.response.status,
    })
  }

  // Hay respuesta pero no tiene la forma esperada: pasa con un 502 de un
  // proxy o con un HTML de error. Se muestra el codigo HTTP, que al menos
  // orienta.
  if (error?.response) {
    return new ErrorAPI({
      codigo: 'RESPUESTA_INESPERADA',
      mensaje: `El servidor respondio con un error (${error.response.status}).`,
      estado: error.response.status,
    })
  }

  return new ErrorAPI({
    codigo: CODIGO_SIN_RED,
    mensaje: 'No se pudo contactar al servidor. Revisa que la API este corriendo.',
  })
}

// esCancelacion sirve para que las pantallas ignoren en silencio los errores de
// una peticion que ellas mismas abortaron al desmontarse.
export function esCancelacion(error) {
  return error?.codigo === CODIGO_CANCELADO
}
