import { useState } from 'react'

import formulario from '../../estilos/formulario.module.css'

import Aviso from '../../componentes/Aviso.jsx'
import Boton from '../../componentes/Boton.jsx'
import Campo from '../../componentes/Campo.jsx'
import Modal from '../../componentes/Modal.jsx'
import { useAccion } from '../../hooks/useAccion.js'
import { aFechaAPI, soloDia } from '../../utiles/formato.js'

// enUnAño propone una fecha limite por defecto. Una meta sin fecha no se puede
// crear —la API la exige— y llegar al formulario con el campo vacio obliga a
// pensar una fecha antes de poder escribir nada.
function enUnAño() {
  const fecha = new Date()
  fecha.setFullYear(fecha.getFullYear() + 1)
  return soloDia(fecha.toISOString())
}

export default function FormularioMeta({ abierto, meta, alGuardar, alCerrar }) {
  const [datos, setDatos] = useState(() => inicial(meta))
  const { ejecutar, ocupado, error } = useAccion(alGuardar)

  function cambiar(campo, valor) {
    setDatos((actual) => ({ ...actual, [campo]: valor }))
  }

  async function enviar(evento) {
    evento.preventDefault()
    const resultado = await ejecutar({
      nombre: datos.nombre.trim(),
      monto_objetivo: Number(datos.monto_objetivo),
      fecha_limite: aFechaAPI(datos.fecha_limite),
      color: datos.color,
      notas: datos.notas.trim(),
      archivada: datos.archivada,
    })
    if (resultado.ok) alCerrar()
  }

  return (
    <Modal
      abierto={abierto}
      titulo={meta ? 'Editar meta' : 'Nueva meta de ahorro'}
      alCerrar={alCerrar}
      pie={
        <>
          <Boton variante="secundario" onClick={alCerrar} disabled={ocupado}>
            Cancelar
          </Boton>
          <Boton type="submit" form="formulario-meta" ocupado={ocupado}>
            Guardar
          </Boton>
        </>
      }
    >
      <form id="formulario-meta" className={formulario.formulario} onSubmit={enviar} noValidate>
        {error && <Aviso tono="error">{error.texto}</Aviso>}

        <Campo
          etiqueta="Nombre"
          value={datos.nombre}
          onChange={(e) => cambiar('nombre', e.target.value)}
          maxLength={80}
          required
          autoFocus
        />

        <div className={formulario.par}>
          <Campo
            etiqueta="Cuanto quieres juntar"
            type="number"
            step="0.01"
            min="0.01"
            value={datos.monto_objetivo}
            onChange={(e) => cambiar('monto_objetivo', e.target.value)}
            required
          />

          <Campo
            etiqueta="Para cuando"
            type="date"
            value={datos.fecha_limite}
            onChange={(e) => cambiar('fecha_limite', e.target.value)}
            ayuda="Es lo que permite calcular el ritmo mensual."
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

        <Campo etiqueta="Notas (opcional)">
          <textarea
            value={datos.notas}
            onChange={(e) => cambiar('notas', e.target.value)}
            maxLength={500}
            rows={2}
          />
        </Campo>

        {/* Editar el objetivo no borra lo ahorrado: las aportaciones siguen
            donde estaban y solo cambia el porcentaje. Se dice aqui porque es la
            duda razonable al subir la cifra. */}
        {meta && (
          <p style={{ fontSize: '0.8rem', color: 'var(--texto-tenue)' }}>
            Cambiar el objetivo no borra lo que ya llevas ahorrado.
          </p>
        )}

        <label className={formulario.acciones} style={{ justifyContent: 'flex-start', gap: 8 }}>
          <input
            type="checkbox"
            checked={datos.archivada}
            onChange={(e) => cambiar('archivada', e.target.checked)}
          />
          Archivada (ya no la estoy persiguiendo)
        </label>
      </form>
    </Modal>
  )
}

function inicial(meta) {
  if (meta) {
    return {
      nombre: meta.nombre,
      monto_objetivo: String(meta.monto_objetivo),
      fecha_limite: soloDia(meta.fecha_limite),
      color: meta.color,
      notas: meta.notas || '',
      archivada: meta.archivada,
    }
  }
  return {
    nombre: '',
    monto_objetivo: '',
    fecha_limite: enUnAño(),
    color: '#0891B2',
    notas: '',
    archivada: false,
  }
}
