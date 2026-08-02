import { useState } from 'react'

import formulario from '../../estilos/formulario.module.css'

import Aviso from '../../componentes/Aviso.jsx'
import Boton from '../../componentes/Boton.jsx'
import Campo from '../../componentes/Campo.jsx'
import Modal from '../../componentes/Modal.jsx'
import { useAccion } from '../../hooks/useAccion.js'

// Los cuatro tipos que acepta el $jsonSchema de MongoDB y la etiqueta binding
// del backend. Si se añade uno hay que tocar los tres sitios.
const TIPOS = [
  { valor: 'efectivo', texto: 'Efectivo' },
  { valor: 'debito', texto: 'Debito' },
  { valor: 'credito', texto: 'Credito' },
  { valor: 'ahorro', texto: 'Ahorro' },
]

const NUEVA = { nombre: '', tipo: 'debito', saldo_inicial: '0', color: '#2563EB', archivada: false }

export default function FormularioCuenta({ abierto, cuenta, alGuardar, alCerrar }) {
  // La clave del modal fuerza a React a montar el formulario de cero cada vez
  // que se abre con otra cuenta; por eso este useState puede leer la cuenta una
  // sola vez y no necesita un efecto que lo sincronice.
  const [datos, setDatos] = useState(() => (cuenta ? aFormulario(cuenta) : NUEVA))
  const { ejecutar, ocupado, error } = useAccion(alGuardar)

  function cambiar(campo, valor) {
    setDatos((actual) => ({ ...actual, [campo]: valor }))
  }

  async function enviar(evento) {
    evento.preventDefault()
    // El saldo se manda como numero: la API lo espera asi y el <input> siempre
    // devuelve cadenas, aunque sea de tipo number.
    const resultado = await ejecutar({ ...datos, saldo_inicial: Number(datos.saldo_inicial) })
    if (resultado.ok) alCerrar()
  }

  return (
    <Modal
      abierto={abierto}
      titulo={cuenta ? 'Editar cuenta' : 'Nueva cuenta'}
      alCerrar={alCerrar}
      pie={
        <>
          <Boton variante="secundario" onClick={alCerrar} disabled={ocupado}>
            Cancelar
          </Boton>
          <Boton type="submit" form="formulario-cuenta" ocupado={ocupado}>
            Guardar
          </Boton>
        </>
      }
    >
      {/* El formulario tiene id porque su boton de envio vive en el pie del
          modal, fuera del <form>. El atributo form los une. */}
      <form id="formulario-cuenta" className={formulario.formulario} onSubmit={enviar} noValidate>
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
          <Campo etiqueta="Tipo">
            <select value={datos.tipo} onChange={(e) => cambiar('tipo', e.target.value)}>
              {TIPOS.map((tipo) => (
                <option key={tipo.valor} value={tipo.valor}>
                  {tipo.texto}
                </option>
              ))}
            </select>
          </Campo>

          <Campo
            etiqueta="Saldo inicial"
            type="number"
            step="0.01"
            value={datos.saldo_inicial}
            onChange={(e) => cambiar('saldo_inicial', e.target.value)}
            ayuda="Puede ser negativo en una tarjeta de credito."
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

        {/* Archivar es lo que se hace con una cuenta que ya no se usa pero que
            tiene movimientos: la API se niega a borrarla para no dejar
            transacciones huerfanas. */}
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

// aFormulario adapta lo que devuelve /reportes/saldos, donde el identificador
// se llama cuenta_id y ademas vienen las sumas calculadas que aqui no se editan.
function aFormulario(cuenta) {
  return {
    nombre: cuenta.nombre,
    tipo: cuenta.tipo,
    saldo_inicial: String(cuenta.saldo_inicial),
    color: cuenta.color,
    archivada: cuenta.archivada,
  }
}
