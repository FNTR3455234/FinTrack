import { useState } from 'react'

import Aviso from './Aviso.jsx'
import Boton from './Boton.jsx'
import Modal from './Modal.jsx'

// ConfirmarBorrado es el paso intermedio de todos los DELETE de la app.
//
// Ademas de preguntar, es donde se enseña el 409 del servidor. La API se niega
// a borrar una cuenta con movimientos o una categoria con presupuestos, y ese
// mensaje tiene que aparecer justo aqui, que es donde el usuario acaba de
// intentarlo, y no como un aviso suelto en otra parte de la pantalla.
export default function ConfirmarBorrado({ abierto, nombre, alCerrar, alConfirmar }) {
  const [borrando, setBorrando] = useState(false)
  const [error, setError] = useState(null)

  function cerrar() {
    if (borrando) return
    setError(null)
    alCerrar()
  }

  async function confirmar() {
    setBorrando(true)
    setError(null)
    try {
      await alConfirmar()
      alCerrar()
    } catch (fallo) {
      setError(fallo)
    } finally {
      setBorrando(false)
    }
  }

  return (
    <Modal
      abierto={abierto}
      titulo="Confirmar borrado"
      alCerrar={cerrar}
      pie={
        <>
          <Boton variante="secundario" onClick={cerrar} disabled={borrando}>
            Cancelar
          </Boton>
          <Boton variante="peligro" onClick={confirmar} ocupado={borrando}>
            Borrar
          </Boton>
        </>
      }
    >
      <p>
        ¿Seguro que quieres borrar <strong>{nombre}</strong>? Esta accion no se puede deshacer.
      </p>
      {error && (
        <div style={{ marginTop: 12 }}>
          <Aviso tono="error" titulo="No se pudo borrar">
            {error.texto}
          </Aviso>
        </div>
      )}
    </Modal>
  )
}
