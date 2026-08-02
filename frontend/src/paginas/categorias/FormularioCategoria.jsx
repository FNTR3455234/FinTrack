import { useState } from 'react'

import formulario from '../../estilos/formulario.module.css'

import Aviso from '../../componentes/Aviso.jsx'
import Boton from '../../componentes/Boton.jsx'
import Campo from '../../componentes/Campo.jsx'
import Modal from '../../componentes/Modal.jsx'
import { useAccion } from '../../hooks/useAccion.js'

const NUEVA = { nombre: '', tipo: 'gasto', color: '#7C3AED', icono: '📁', archivada: false }

export default function FormularioCategoria({ abierto, categoria, alGuardar, alCerrar }) {
  const [datos, setDatos] = useState(() => categoria ?? NUEVA)
  const { ejecutar, ocupado, error } = useAccion(alGuardar)

  function cambiar(campo, valor) {
    setDatos((actual) => ({ ...actual, [campo]: valor }))
  }

  async function enviar(evento) {
    evento.preventDefault()
    const resultado = await ejecutar({
      nombre: datos.nombre,
      tipo: datos.tipo,
      color: datos.color,
      icono: datos.icono,
      archivada: datos.archivada,
    })
    if (resultado.ok) alCerrar()
  }

  return (
    <Modal
      abierto={abierto}
      titulo={categoria ? 'Editar categoria' : 'Nueva categoria'}
      alCerrar={alCerrar}
      pie={
        <>
          <Boton variante="secundario" onClick={alCerrar} disabled={ocupado}>
            Cancelar
          </Boton>
          <Boton type="submit" form="formulario-categoria" ocupado={ocupado}>
            Guardar
          </Boton>
        </>
      }
    >
      <form id="formulario-categoria" className={formulario.formulario} onSubmit={enviar} noValidate>
        {error && <Aviso tono="error">{error.texto}</Aviso>}

        <Campo
          etiqueta="Nombre"
          value={datos.nombre}
          onChange={(e) => cambiar('nombre', e.target.value)}
          maxLength={60}
          required
          autoFocus
        />

        <div className={formulario.par}>
          <Campo
            etiqueta="Tipo"
            // Cambiar el tipo de una categoria que ya tiene movimientos dejaria
            // gastos colgando de una categoria de ingreso, y el servidor lo
            // rechazaria al siguiente guardado de esas transacciones. Se avisa
            // aqui en vez de dejar que el error salga mucho despues.
            ayuda={categoria ? 'Cambiarlo afecta a los movimientos que ya la usan.' : undefined}
          >
            <select value={datos.tipo} onChange={(e) => cambiar('tipo', e.target.value)}>
              <option value="gasto">Gasto</option>
              <option value="ingreso">Ingreso</option>
            </select>
          </Campo>

          <Campo
            etiqueta="Icono"
            value={datos.icono}
            onChange={(e) => cambiar('icono', e.target.value)}
            ayuda="Un emoji, por ejemplo 🛒"
            maxLength={40}
            required
          />
        </div>

        <Campo etiqueta="Color">
          <input
            type="color"
            value={datos.color}
            onChange={(e) => cambiar('color', e.target.value.toUpperCase())}
          />
        </Campo>

        <label className={formulario.acciones} style={{ justifyContent: 'flex-start', gap: 8 }}>
          <input
            type="checkbox"
            checked={datos.archivada}
            onChange={(e) => cambiar('archivada', e.target.checked)}
          />
          Archivada (no aparece al registrar movimientos)
        </label>
      </form>
    </Modal>
  )
}
