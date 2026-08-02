import { Link } from 'react-router-dom'

import { Vacio } from '../componentes/Estados.jsx'

// Pantalla para una direccion que no existe. Es la unica ruta publica que no es
// un formulario: si alguien con sesion escribe mal la URL, el enlace lo devuelve
// al tablero.
export default function NoEncontrada() {
  return (
    <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center', padding: 24 }}>
      <Vacio titulo="Esta pagina no existe" accion={<Link to="/">Volver al tablero</Link>}>
        Puede que el enlace este mal escrito o que la seccion se haya movido.
      </Vacio>
    </div>
  )
}
