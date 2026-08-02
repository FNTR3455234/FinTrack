import { useEffect } from 'react'

import estilos from './Encabezado.module.css'

// Encabezado es el titulo de cada pagina con su acción principal a la derecha.
//
// De paso escribe el <title> del documento. Sin eso todas las pestañas se
// llamarian "FinTrack" y el historial del navegador seria inservible: en una
// aplicacion de una sola pagina, el titulo no cambia solo.
export default function Encabezado({ titulo, descripcion, children }) {
  useEffect(() => {
    document.title = `${titulo} · FinTrack`
  }, [titulo])

  return (
    <header className={estilos.encabezado}>
      <div>
        <h1>{titulo}</h1>
        {descripcion && <p className={estilos.descripcion}>{descripcion}</p>}
      </div>
      {children && <div className={estilos.acciones}>{children}</div>}
    </header>
  )
}
