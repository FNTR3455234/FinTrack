import { useCallback, useState } from 'react'

import estilos from './Presupuestos.module.css'

import { AvisoDeError } from '../componentes/Aviso.jsx'
import BarraProgreso from '../componentes/BarraProgreso.jsx'
import Boton from '../componentes/Boton.jsx'
import ConfirmarBorrado from '../componentes/ConfirmarBorrado.jsx'
import Encabezado from '../componentes/Encabezado.jsx'
import { Cargando, Vacio } from '../componentes/Estados.jsx'
import { EstadoPresupuesto } from '../componentes/Etiqueta.jsx'
import SelectorPeriodo from '../componentes/SelectorPeriodo.jsx'
import Tarjeta from '../componentes/Tarjeta.jsx'
import FormularioPresupuesto from './presupuestos/FormularioPresupuesto.jsx'
import { categorias as apiCategorias, presupuestos as api } from '../api/catalogos.js'
import * as reportes from '../api/reportes.js'
import { usePeriodo } from '../hooks/usePeriodo.js'
import { usePeticion } from '../hooks/usePeticion.js'
import { periodoLargo } from '../utiles/formato.js'

// La pantalla lee de /reportes/estado-presupuestos y no de /presupuestos.
//
// El listado a secas devolveria solo el limite; el reporte devuelve ademas lo
// gastado y el semaforo, que es justo lo que se viene a ver aqui. Y es la misma
// consulta que alimenta el tablero y la alerta al registrar un gasto: asi las
// tres pantallas no pueden decir cosas distintas.
export default function Presupuestos() {
  const { periodo, mover, alMesActual, esMesActual } = usePeriodo()
  const [editando, setEditando] = useState(null)
  const [borrando, setBorrando] = useState(null)

  const cargar = useCallback(
    ({ signal }) =>
      Promise.all([
        reportes.estadoPresupuestos(periodo, { signal }),
        apiCategorias.listar({ tipo: 'gasto' }, { signal }),
      ]).then(([estados, categorias]) => ({ estados, categorias })),
    [periodo],
  )

  const { datos, cargando, error, recargar } = usePeticion(cargar, [periodo.mes, periodo.anio])

  async function guardar(valores) {
    if (editando.presupuesto_id) {
      await api.actualizar(editando.presupuesto_id, valores)
    } else {
      await api.crear(valores)
    }
    recargar()
  }

  return (
    <>
      <Encabezado
        titulo="Presupuestos"
        descripcion={`Cuanto te propusiste gastar en ${periodoLargo(periodo)}.`}
      >
        <SelectorPeriodo
          periodo={periodo}
          mover={mover}
          alMesActual={alMesActual}
          esMesActual={esMesActual}
        />
        <Boton onClick={() => setEditando({})} disabled={!datos?.categorias.length}>
          Nuevo presupuesto
        </Boton>
      </Encabezado>

      {error && <AvisoDeError error={error} alReintentar={recargar} />}

      <Tarjeta>
        {cargando && !datos && <Cargando etiqueta="Cargando los presupuestos" filas={4} />}

        {datos && datos.categorias.length === 0 && (
          <Vacio titulo="Antes necesitas una categoria de gasto">
            Un presupuesto es el limite de una categoria, asi que primero hay que crear la
            categoria.
          </Vacio>
        )}

        {datos && datos.categorias.length > 0 && datos.estados.length === 0 && (
          <Vacio
            titulo={`Sin presupuestos en ${periodoLargo(periodo)}`}
            accion={<Boton onClick={() => setEditando({})}>Crear uno</Boton>}
          >
            Los presupuestos son de un mes concreto: los de un mes no se arrastran al
            siguiente.
          </Vacio>
        )}

        {datos && datos.estados.length > 0 && (
          <ul className={estilos.lista} data-cargando={cargando || undefined}>
            {datos.estados.map((estado) => (
              <li key={estado.presupuesto_id} className={estilos.fila}>
                <div className={estilos.barra}>
                  <BarraProgreso estado={estado} />
                </div>
                <div className={estilos.acciones}>
                  <EstadoPresupuesto estado={estado.estado} />
                  <Boton variante="texto" onClick={() => setEditando(estado)}>
                    Editar
                  </Boton>
                  <Boton variante="texto" onClick={() => setBorrando(estado)}>
                    Borrar
                  </Boton>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Tarjeta>

      {editando && datos && (
        <FormularioPresupuesto
          key={editando.presupuesto_id || 'nuevo'}
          abierto
          presupuesto={editando.presupuesto_id ? editando : null}
          categorias={datos.categorias}
          periodo={periodo}
          alGuardar={guardar}
          alCerrar={() => setEditando(null)}
        />
      )}

      {borrando && (
        <ConfirmarBorrado
          abierto
          nombre={`el presupuesto de ${borrando.nombre}`}
          alCerrar={() => setBorrando(null)}
          alConfirmar={() => api.eliminar(borrando.presupuesto_id).then(recargar)}
        />
      )}
    </>
  )
}
