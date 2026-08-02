import estilos from './Paginacion.module.css'

// Paginacion pinta la meta que devuelve el listado de transacciones:
// { pagina, limite, total, total_paginas }.
//
// Se muestra el total de registros y no solo el numero de pagina porque es el
// dato que la gente busca de verdad ("¿cuantos movimientos tengo en julio?").
export default function Paginacion({ meta, alCambiar }) {
  if (!meta || meta.total_paginas <= 1) {
    return meta ? <p className={estilos.resumen}>{textoTotal(meta)}</p> : null
  }

  return (
    <nav className={estilos.paginacion} aria-label="Paginacion del listado">
      <p className={estilos.resumen}>{textoTotal(meta)}</p>

      <div className={estilos.controles}>
        <button
          type="button"
          className={estilos.boton}
          onClick={() => alCambiar(meta.pagina - 1)}
          disabled={meta.pagina <= 1}
        >
          Anterior
        </button>

        <span className={estilos.actual} aria-live="polite">
          Pagina {meta.pagina} de {meta.total_paginas}
        </span>

        <button
          type="button"
          className={estilos.boton}
          onClick={() => alCambiar(meta.pagina + 1)}
          disabled={meta.pagina >= meta.total_paginas}
        >
          Siguiente
        </button>
      </div>
    </nav>
  )
}

function textoTotal({ total }) {
  return total === 1 ? '1 movimiento' : `${total} movimientos`
}
