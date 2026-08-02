import { useMemo, useState } from 'react'

import formulario from '../../estilos/formulario.module.css'

import Aviso from '../../componentes/Aviso.jsx'
import Boton from '../../componentes/Boton.jsx'
import Campo from '../../componentes/Campo.jsx'
import Modal from '../../componentes/Modal.jsx'
import { useAccion } from '../../hooks/useAccion.js'
import { aFechaAPI, hoy, soloDia } from '../../utiles/formato.js'

// Formulario de alta y edicion de un movimiento.
//
// La regla que manda aqui es que el tipo del movimiento y el de su categoria
// tienen que coincidir: el servidor rechaza un gasto en una categoria de
// ingreso con 422 TIPO_NO_COINCIDE. Por eso el desplegable de categorias solo
// muestra las del tipo elegido, y al cambiar el tipo se limpia la categoria en
// vez de dejar una que ya no vale.
export default function FormularioTransaccion({
  abierto,
  transaccion,
  cuentas,
  categorias,
  alGuardar,
  alCerrar,
}) {
  const [datos, setDatos] = useState(() => inicial(transaccion, cuentas))
  const { ejecutar, ocupado, error } = useAccion(alGuardar)

  const categoriasDelTipo = useMemo(
    () => categorias.filter((categoria) => categoria.tipo === datos.tipo),
    [categorias, datos.tipo],
  )

  function cambiar(campo, valor) {
    setDatos((actual) => ({ ...actual, [campo]: valor }))
  }

  function cambiarTipo(tipo) {
    setDatos((actual) => ({ ...actual, tipo, categoria_id: '' }))
  }

  async function enviar(evento) {
    evento.preventDefault()
    const resultado = await ejecutar({
      cuenta_id: datos.cuenta_id,
      categoria_id: datos.categoria_id,
      tipo: datos.tipo,
      monto: Number(datos.monto),
      descripcion: datos.descripcion.trim(),
      // La API espera un instante completo, no un dia suelto: las 12:00 UTC son
      // el ancla que usa el backend para que el dia no se corra al cambiar de
      // huso horario.
      fecha: aFechaAPI(datos.fecha),
      notas: datos.notas.trim(),
    })
    if (resultado.ok) alCerrar()
  }

  return (
    <Modal
      abierto={abierto}
      titulo={transaccion ? 'Editar movimiento' : 'Nuevo movimiento'}
      alCerrar={alCerrar}
      pie={
        <>
          <Boton variante="secundario" onClick={alCerrar} disabled={ocupado}>
            Cancelar
          </Boton>
          <Boton type="submit" form="formulario-transaccion" ocupado={ocupado}>
            Guardar
          </Boton>
        </>
      }
    >
      <form
        id="formulario-transaccion"
        className={formulario.formulario}
        onSubmit={enviar}
        noValidate
      >
        {error && <Aviso tono="error">{error.texto}</Aviso>}

        <div className={formulario.par}>
          <Campo etiqueta="Tipo">
            <select value={datos.tipo} onChange={(e) => cambiarTipo(e.target.value)}>
              <option value="gasto">Gasto</option>
              <option value="ingreso">Ingreso</option>
            </select>
          </Campo>

          {/* El monto siempre es positivo: lo que decide si suma o resta es el
              tipo. Por eso el minimo es 0.01 y no hay signo que elegir. */}
          <Campo
            etiqueta="Monto"
            type="number"
            step="0.01"
            min="0.01"
            value={datos.monto}
            onChange={(e) => cambiar('monto', e.target.value)}
            required
            autoFocus
          />
        </div>

        <Campo
          etiqueta="Descripcion"
          value={datos.descripcion}
          onChange={(e) => cambiar('descripcion', e.target.value)}
          maxLength={140}
          required
        />

        <div className={formulario.par}>
          <Campo etiqueta="Cuenta">
            <select
              value={datos.cuenta_id}
              onChange={(e) => cambiar('cuenta_id', e.target.value)}
              required
            >
              <option value="">Elige una cuenta</option>
              {cuentas.map((cuenta) => (
                <option key={cuenta.id} value={cuenta.id}>
                  {cuenta.nombre}
                </option>
              ))}
            </select>
          </Campo>

          <Campo
            etiqueta="Categoria"
            ayuda={
              categoriasDelTipo.length === 0
                ? `No tienes categorias de ${datos.tipo}.`
                : undefined
            }
          >
            <select
              value={datos.categoria_id}
              onChange={(e) => cambiar('categoria_id', e.target.value)}
              required
            >
              <option value="">Elige una categoria</option>
              {categoriasDelTipo.map((categoria) => (
                <option key={categoria.id} value={categoria.id}>
                  {categoria.nombre}
                </option>
              ))}
            </select>
          </Campo>
        </div>

        <Campo
          etiqueta="Fecha"
          type="date"
          value={datos.fecha}
          onChange={(e) => cambiar('fecha', e.target.value)}
          required
        />

        <Campo etiqueta="Notas (opcional)">
          <textarea
            value={datos.notas}
            onChange={(e) => cambiar('notas', e.target.value)}
            maxLength={500}
            rows={3}
          />
        </Campo>
      </form>
    </Modal>
  )
}

// inicial arranca un movimiento nuevo como gasto de hoy en la primera cuenta:
// es con diferencia lo que mas se registra, y ahorra tres clics en el caso
// normal.
function inicial(transaccion, cuentas) {
  if (transaccion) {
    return {
      cuenta_id: transaccion.cuenta_id,
      categoria_id: transaccion.categoria_id,
      tipo: transaccion.tipo,
      monto: String(transaccion.monto),
      descripcion: transaccion.descripcion,
      fecha: soloDia(transaccion.fecha),
      notas: transaccion.notas || '',
    }
  }
  return {
    cuenta_id: cuentas[0]?.id || '',
    categoria_id: '',
    tipo: 'gasto',
    monto: '',
    descripcion: '',
    fecha: hoy(),
    notas: '',
  }
}
