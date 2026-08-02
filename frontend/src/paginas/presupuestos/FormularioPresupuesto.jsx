import { useState } from 'react'

import formulario from '../../estilos/formulario.module.css'

import Aviso from '../../componentes/Aviso.jsx'
import Boton from '../../componentes/Boton.jsx'
import Campo from '../../componentes/Campo.jsx'
import Modal from '../../componentes/Modal.jsx'
import { useAccion } from '../../hooks/useAccion.js'
import { MESES } from '../../utiles/formato.js'

// Solo se presupuestan gastos: ponerle un techo a un ingreso no significa nada.
// La API lo comprueba tambien, asi que aqui el filtro es una comodidad, no la
// defensa.
export default function FormularioPresupuesto({
  abierto,
  presupuesto,
  categorias,
  periodo,
  alGuardar,
  alCerrar,
}) {
  const [datos, setDatos] = useState(() => inicial(presupuesto, categorias, periodo))
  const { ejecutar, ocupado, error } = useAccion(alGuardar)

  function cambiar(campo, valor) {
    setDatos((actual) => ({ ...actual, [campo]: valor }))
  }

  async function enviar(evento) {
    evento.preventDefault()
    const resultado = await ejecutar({
      categoria_id: datos.categoria_id,
      monto_limite: Number(datos.monto_limite),
      mes: Number(datos.mes),
      anio: Number(datos.anio),
    })
    if (resultado.ok) alCerrar()
  }

  return (
    <Modal
      abierto={abierto}
      titulo={presupuesto ? 'Editar presupuesto' : 'Nuevo presupuesto'}
      alCerrar={alCerrar}
      pie={
        <>
          <Boton variante="secundario" onClick={alCerrar} disabled={ocupado}>
            Cancelar
          </Boton>
          <Boton type="submit" form="formulario-presupuesto" ocupado={ocupado}>
            Guardar
          </Boton>
        </>
      }
    >
      <form
        id="formulario-presupuesto"
        className={formulario.formulario}
        onSubmit={enviar}
        noValidate
      >
        {/* Aqui es donde aparece el 409 PRESUPUESTO_DUPLICADO. No se comprueba
            antes en el cliente a proposito: quien decide si ya existe es el
            indice unico de MongoDB, y preguntar primero solo abriria una ventana
            entre la pregunta y la escritura. */}
        {error && <Aviso tono="error">{error.texto}</Aviso>}

        <Campo etiqueta="Categoria">
          <select
            value={datos.categoria_id}
            onChange={(e) => cambiar('categoria_id', e.target.value)}
            required
          >
            <option value="">Elige una categoria de gasto</option>
            {categorias.map((categoria) => (
              <option key={categoria.id} value={categoria.id}>
                {categoria.nombre}
              </option>
            ))}
          </select>
        </Campo>

        <Campo
          etiqueta="Limite del mes"
          type="number"
          step="0.01"
          min="0.01"
          value={datos.monto_limite}
          onChange={(e) => cambiar('monto_limite', e.target.value)}
          required
        />

        <div className={formulario.par}>
          <Campo etiqueta="Mes">
            <select value={datos.mes} onChange={(e) => cambiar('mes', e.target.value)}>
              {MESES.map((nombre, indice) => (
                <option key={nombre} value={indice + 1}>
                  {nombre}
                </option>
              ))}
            </select>
          </Campo>

          <Campo
            etiqueta="Año"
            type="number"
            min="2000"
            max="2100"
            value={datos.anio}
            onChange={(e) => cambiar('anio', e.target.value)}
            required
          />
        </div>
      </form>
    </Modal>
  )
}

// inicial arranca el formulario en el periodo que se esta mirando: casi siempre
// el presupuesto que se quiere crear es el del mes que hay en pantalla.
function inicial(presupuesto, categorias, periodo) {
  if (presupuesto) {
    return {
      categoria_id: presupuesto.categoria_id,
      monto_limite: String(presupuesto.monto_limite),
      mes: presupuesto.mes ?? periodo.mes,
      anio: presupuesto.anio ?? periodo.anio,
    }
  }
  return {
    categoria_id: categorias[0]?.id || '',
    monto_limite: '',
    mes: periodo.mes,
    anio: periodo.anio,
  }
}
