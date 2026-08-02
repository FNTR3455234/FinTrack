import estilos from './Aviso.module.css'

// Aviso es el recuadro de mensajes: errores, advertencias y confirmaciones.
//
// El tono cambia el color, pero el texto siempre dice lo que pasa: quien no
// distingue el rojo del ambar tiene que poder enterarse igual.
//
// role="alert" solo en los errores. Si todos los avisos lo llevaran, el lector
// de pantalla interrumpiria lo que este leyendo tambien para un "Guardado", y
// eso acaba siendo ruido que la gente aprende a ignorar.
export default function Aviso({ tono = 'error', titulo, children, accion }) {
  return (
    <div
      className={estilos.aviso}
      data-tono={tono}
      role={tono === 'error' ? 'alert' : 'status'}
    >
      <div className={estilos.texto}>
        {titulo && <strong className={estilos.titulo}>{titulo}</strong>}
        {children && <div>{children}</div>}
      </div>
      {accion && <div className={estilos.accion}>{accion}</div>}
    </div>
  )
}

// AvisoDeError es el atajo para el caso mas repetido: mostrar un ErrorAPI con
// un boton de reintentar.
export function AvisoDeError({ error, alReintentar }) {
  if (!error) return null
  return (
    <Aviso
      tono="error"
      titulo="No se pudo cargar"
      accion={
        alReintentar && (
          <button type="button" className={estilos.enlace} onClick={alReintentar}>
            Reintentar
          </button>
        )
      }
    >
      {error.texto}
    </Aviso>
  )
}
