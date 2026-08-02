import { cloneElement, useId } from 'react'

import estilos from './Campo.module.css'

// Campo es una etiqueta, un control y su mensaje de error.
//
// El useId no es adorno: sin un id que una la <label> con su control, hacer clic
// en la etiqueta no enfoca el campo y un lector de pantalla anuncia "cuadro de
// texto" sin decir de que. Generarlo aqui evita inventar ids unicos a mano en
// cada formulario.
//
// Si no se le pasa un hijo, Campo pinta un <input> con las props que sobren; si
// se le pasa uno (un select, un textarea), le inyecta los mismos atributos con
// cloneElement. Asi todos los controles de la app se comportan igual.
export default function Campo({ etiqueta, error, ayuda, children, ...resto }) {
  const id = useId()
  const idError = `${id}-error`
  const idAyuda = `${id}-ayuda`

  // aria-describedby apunta al error o a la ayuda para que el lector los lea
  // junto con el campo y no como texto suelto en otra parte de la pagina.
  const descrito = [error ? idError : null, ayuda ? idAyuda : null].filter(Boolean).join(' ')

  const comunes = {
    id,
    'aria-describedby': descrito || undefined,
    'aria-invalid': error ? true : undefined,
    'data-error': error ? '' : undefined,
  }

  const control = children ? (
    cloneElement(children, {
      ...comunes,
      className: [estilos.control, children.props.className].filter(Boolean).join(' '),
    })
  ) : (
    <input className={estilos.control} {...comunes} {...resto} />
  )

  return (
    <div className={estilos.campo}>
      <label className={estilos.etiqueta} htmlFor={id}>
        {etiqueta}
      </label>

      {control}

      {ayuda && !error && (
        <p className={estilos.ayuda} id={idAyuda}>
          {ayuda}
        </p>
      )}
      {error && (
        // role="alert" hace que el lector lo anuncie en cuanto aparece, sin
        // esperar a que el usuario vuelva a recorrer el formulario.
        <p className={estilos.error} id={idError} role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
