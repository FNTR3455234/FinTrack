import { useEffect, useRef } from 'react'

import estilos from './Modal.module.css'

// Modal usa el <dialog> nativo en vez de un <div> con position:fixed.
//
// showModal() trae hechas tres cosas que a mano cuestan bastante y que casi
// siempre acaban mal: el foco queda atrapado dentro del dialogo, Escape lo
// cierra, y el resto de la pagina queda inerte para el raton y para los
// lectores de pantalla. Escribir eso a mano seria mas codigo y peor.
export default function Modal({ abierto, titulo, alCerrar, children, pie }) {
  const dialogo = useRef(null)

  useEffect(() => {
    const elemento = dialogo.current
    if (!elemento) return

    if (abierto && !elemento.open) {
      elemento.showModal()
    } else if (!abierto && elemento.open) {
      elemento.close()
    }
  }, [abierto])

  // El evento close lo dispara tanto el boton de cerrar como la tecla Escape.
  // Escuchandolo aqui, el estado de React no se queda desincronizado cuando el
  // usuario cierra con el teclado.
  useEffect(() => {
    const elemento = dialogo.current
    if (!elemento) return

    const alCerrarse = () => alCerrar()
    elemento.addEventListener('close', alCerrarse)
    return () => elemento.removeEventListener('close', alCerrarse)
  }, [alCerrar])

  // Clic fuera del contenido: el <dialog> ocupa toda la pantalla y su fondo es
  // el propio elemento, asi que un clic cuyo destino es el dialogo (y no algo
  // de dentro) es un clic en el fondo.
  function alHacerClic(evento) {
    if (evento.target === dialogo.current) alCerrar()
  }

  return (
    <dialog ref={dialogo} className={estilos.dialogo} onClick={alHacerClic} aria-label={titulo}>
      <div className={estilos.caja}>
        <header className={estilos.cabecera}>
          <h2 className={estilos.titulo}>{titulo}</h2>
          <button type="button" className={estilos.cerrar} onClick={alCerrar}>
            <span aria-hidden="true">×</span>
            <span className="solo-lectores">Cerrar</span>
          </button>
        </header>

        <div className={estilos.cuerpo}>{children}</div>

        {pie && <footer className={estilos.pie}>{pie}</footer>}
      </div>
    </dialog>
  )
}
