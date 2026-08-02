import estilos from './SelectorPeriodo.module.css'

import { periodoLargo } from '../utiles/formato.js'

// SelectorPeriodo son las flechas de mes anterior y mes siguiente con el
// periodo escrito en medio.
//
// El texto va en un aria-live: quien navega con lector de pantalla pulsa la
// flecha y necesita oir a que mes acaba de llegar, porque el resto de la
// pantalla cambia sin avisar.
export default function SelectorPeriodo({ periodo, mover, alMesActual, esMesActual }) {
  return (
    <div className={estilos.selector}>
      <button type="button" className={estilos.flecha} onClick={() => mover(-1)}>
        <span aria-hidden="true">‹</span>
        <span className="solo-lectores">Mes anterior</span>
      </button>

      <span className={estilos.periodo} aria-live="polite">
        {periodoLargo(periodo)}
      </span>

      <button type="button" className={estilos.flecha} onClick={() => mover(1)}>
        <span aria-hidden="true">›</span>
        <span className="solo-lectores">Mes siguiente</span>
      </button>

      <button
        type="button"
        className={estilos.hoy}
        onClick={alMesActual}
        disabled={esMesActual}
      >
        Hoy
      </button>
    </div>
  )
}
