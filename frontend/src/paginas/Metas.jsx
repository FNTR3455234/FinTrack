import { useCallback, useState } from 'react'

import estilos from './Metas.module.css'

import { AvisoDeError } from '../componentes/Aviso.jsx'
import BarraMeta from '../componentes/BarraMeta.jsx'
import Boton from '../componentes/Boton.jsx'
import ConfirmarBorrado from '../componentes/ConfirmarBorrado.jsx'
import Encabezado from '../componentes/Encabezado.jsx'
import { Cargando, Vacio } from '../componentes/Estados.jsx'
import { EstadoMeta } from '../componentes/Etiqueta.jsx'
import Tarjeta, { Cifra } from '../componentes/Tarjeta.jsx'
import FormularioMeta from './metas/FormularioMeta.jsx'
import PanelAportaciones from './metas/PanelAportaciones.jsx'
import * as api from '../api/metas.js'
import { useAuth } from '../contexto/AuthContexto.jsx'
import { usePeticion } from '../hooks/usePeticion.js'
import { dinero, porcentaje } from '../utiles/formato.js'

export default function Metas() {
  const { usuario } = useAuth()
  const [verArchivadas, setVerArchivadas] = useState(false)
  const [editando, setEditando] = useState(null)
  const [borrando, setBorrando] = useState(null)
  const [abierta, setAbierta] = useState(null)

  const cargar = useCallback(
    ({ signal }) =>
      api.listar(verArchivadas ? { incluir_archivadas: true } : undefined, { signal }),
    [verArchivadas],
  )
  const { datos, cargando, error, recargar } = usePeticion(cargar, [verArchivadas])

  async function guardar(valores) {
    if (editando.meta_id) {
      await api.actualizar(editando.meta_id, valores)
    } else {
      await api.crear(valores)
    }
    recargar()
  }

  return (
    <>
      <Encabezado
        titulo="Metas de ahorro"
        descripcion="Cuanto quieres juntar, para cuando y a que ritmo tendrias que ir."
      >
        <label className={estilos.interruptor}>
          <input
            type="checkbox"
            checked={verArchivadas}
            onChange={(evento) => setVerArchivadas(evento.target.checked)}
          />
          Ver archivadas
        </label>
        <Boton onClick={() => setEditando({})}>Nueva meta</Boton>
      </Encabezado>

      {error && <AvisoDeError error={error} alReintentar={recargar} />}
      {!datos && cargando && <Cargando etiqueta="Cargando las metas" filas={5} />}

      {datos && (
        <div className={estilos.pila} data-cargando={cargando || undefined}>
          {datos.metas.length > 0 && <Resumen resumen={datos.resumen} moneda={usuario.moneda} />}

          <Tarjeta>
            {datos.metas.length === 0 ? (
              <Vacio
                titulo={verArchivadas ? 'No tienes ninguna meta' : 'Todavia no tienes metas activas'}
                accion={<Boton onClick={() => setEditando({})}>Crear la primera</Boton>}
              >
                Una meta es un objetivo con fecha: cuanto quieres juntar y para cuando. Con eso la
                app puede decirte cuanto tendrias que apartar cada mes.
              </Vacio>
            ) : (
              <ul className={estilos.lista}>
                {datos.metas.map((meta) => (
                  <li key={meta.meta_id} className={estilos.fila}>
                    <div className={estilos.barra}>
                      <BarraMeta meta={meta} moneda={usuario.moneda} />
                    </div>
                    <div className={estilos.acciones}>
                      <EstadoMeta estado={meta.estado} />
                      <Boton variante="texto" onClick={() => setAbierta(meta)}>
                        Aportaciones ({meta.aportaciones})
                      </Boton>
                      <Boton variante="texto" onClick={() => setEditando(meta)}>
                        Editar
                      </Boton>
                      <Boton variante="texto" onClick={() => setBorrando(meta)}>
                        Borrar
                      </Boton>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </Tarjeta>
        </div>
      )}

      {editando && (
        <FormularioMeta
          key={editando.meta_id || 'nueva'}
          abierto
          meta={editando.meta_id ? editando : null}
          alGuardar={guardar}
          alCerrar={() => setEditando(null)}
        />
      )}

      {abierta && (
        <PanelAportaciones
          metaID={abierta.meta_id}
          moneda={usuario.moneda}
          alCambiar={recargar}
          alCerrar={() => setAbierta(null)}
        />
      )}

      {borrando && (
        <ConfirmarBorrado
          abierto
          nombre={textoDelBorrado(borrando)}
          alCerrar={() => setBorrando(null)}
          alConfirmar={() => api.eliminar(borrando.meta_id).then(recargar)}
        />
      )}
    </>
  )
}

// Resumen son las tres cifras del conjunto. Las calcula la API, no el cliente:
// asi el numero es el mismo aunque un dia se pagine el listado.
function Resumen({ resumen, moneda }) {
  const avance = resumen.objetivo > 0 ? (resumen.ahorrado / resumen.objetivo) * 100 : 0

  return (
    <div className={estilos.cifras}>
      <Cifra
        rotulo="Ahorrado"
        valor={dinero(resumen.ahorrado, moneda)}
        tono="ingreso"
        pie={`${porcentaje(avance)} de ${dinero(resumen.objetivo, moneda)}`}
      />
      <Cifra rotulo="Metas activas" valor={String(resumen.total)} />
      <Cifra
        rotulo="Cumplidas"
        valor={String(resumen.cumplidas)}
        pie={resumen.vencidas > 0 ? `${resumen.vencidas} vencida(s)` : 'ninguna vencida'}
      />
    </div>
  )
}

// El aviso del borrado dice cuantas aportaciones se van con la meta: es la
// consecuencia que no se ve, y es la unica que no se puede deshacer.
function textoDelBorrado(meta) {
  if (meta.aportaciones === 0) return `la meta "${meta.nombre}"`
  return `la meta "${meta.nombre}" y sus ${meta.aportaciones} aportacion(es)`
}
