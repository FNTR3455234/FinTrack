import { useCallback, useState } from 'react'

import estilos from './PanelAportaciones.module.css'
import formulario from '../../estilos/formulario.module.css'

import Aviso, { AvisoDeError } from '../../componentes/Aviso.jsx'
import BarraMeta from '../../componentes/BarraMeta.jsx'
import Boton from '../../componentes/Boton.jsx'
import Campo from '../../componentes/Campo.jsx'
import { Cargando, Vacio } from '../../componentes/Estados.jsx'
import Modal from '../../componentes/Modal.jsx'
import * as api from '../../api/metas.js'
import { useAccion } from '../../hooks/useAccion.js'
import { usePeticion } from '../../hooks/usePeticion.js'
import { aFechaAPI, dinero, fechaCorta, hoy } from '../../utiles/formato.js'

// PanelAportaciones es el detalle de una meta: la barra, el formulario para
// apartar dinero y la lista de lo ya apartado.
//
// La lista no es decoracion: es lo que explica de donde sale el total. Sin ella
// el usuario ve una cifra y tiene que creersela.
export default function PanelAportaciones({ metaID, moneda, alCerrar, alCambiar }) {
  const [nueva, setNueva] = useState({ monto: '', fecha: hoy(), nota: '' })

  const cargar = useCallback(({ signal }) => api.obtener(metaID, { signal }), [metaID])
  const { datos, cargando, error, recargar } = usePeticion(cargar, [metaID])

  // Al guardar o borrar hay que refrescar dos cosas: este panel y el listado de
  // atras, que tiene su propia copia del progreso.
  const refrescar = useCallback(() => {
    recargar()
    alCambiar()
  }, [recargar, alCambiar])

  const aportacion = useAccion(async () => {
    await api.aportar(metaID, {
      monto: Number(nueva.monto),
      fecha: aFechaAPI(nueva.fecha),
      nota: nueva.nota.trim(),
    })
    setNueva({ monto: '', fecha: hoy(), nota: '' })
    refrescar()
  })

  const borrado = useAccion(async (id) => {
    await api.quitarAportacion(metaID, id)
    refrescar()
  })

  return (
    <Modal abierto titulo={datos?.nombre || 'Meta'} alCerrar={alCerrar}>
      {error && <AvisoDeError error={error} alReintentar={recargar} />}
      {!datos && cargando && <Cargando etiqueta="Cargando la meta" filas={4} />}

      {datos && (
        <div className={estilos.panel} data-cargando={cargando || undefined}>
          <BarraMeta meta={datos} moneda={moneda} />

          {datos.notas && <p className={estilos.notas}>{datos.notas}</p>}

          <form
            className={estilos.alta}
            onSubmit={(evento) => {
              evento.preventDefault()
              aportacion.ejecutar()
            }}
            noValidate
          >
            <Campo
              etiqueta="Apartar"
              type="number"
              step="0.01"
              min="0.01"
              value={nueva.monto}
              onChange={(e) => setNueva({ ...nueva, monto: e.target.value })}
              required
            />
            <Campo
              etiqueta="Fecha"
              type="date"
              value={nueva.fecha}
              onChange={(e) => setNueva({ ...nueva, fecha: e.target.value })}
              required
            />
            <Campo
              etiqueta="Nota"
              value={nueva.nota}
              onChange={(e) => setNueva({ ...nueva, nota: e.target.value })}
              maxLength={140}
            />
            <Boton type="submit" ocupado={aportacion.ocupado}>
              Añadir
            </Boton>
          </form>

          {aportacion.error && <Aviso tono="error">{aportacion.error.texto}</Aviso>}
          {borrado.error && <Aviso tono="error">{borrado.error.texto}</Aviso>}

          {/* Apartar dinero NO es gastarlo: no toca el saldo de ninguna cuenta
              ni cuenta como gasto del mes. Se dice aqui, que es donde se
              registra, y no en un README que nadie va a leer. */}
          <p className={estilos.nota}>
            Apartar dinero para una meta no lo descuenta de tus cuentas ni cuenta como gasto del
            mes: es un registro de lo que has ido separando.
          </p>

          <Lista
            aportaciones={datos.detalle}
            moneda={moneda}
            alBorrar={borrado.ejecutar}
            ocupado={borrado.ocupado}
          />
        </div>
      )}
    </Modal>
  )
}

// Lista pinta las aportaciones, de la mas reciente a la mas vieja.
function Lista({ aportaciones, moneda, alBorrar, ocupado }) {
  if (aportaciones.length === 0) {
    return <Vacio titulo="Todavia no has apartado nada">Registra la primera aportacion arriba.</Vacio>
  }

  return (
    <ul className={estilos.lista}>
      {aportaciones.map((a) => (
        <li key={a.id} className={estilos.fila}>
          <div className={estilos.datos}>
            <span className={`${estilos.monto} dinero`}>{dinero(a.monto, moneda)}</span>
            <span className={estilos.fecha}>{fechaCorta(a.fecha)}</span>
            {a.nota && <span className={estilos.notaFila}>{a.nota}</span>}
          </div>
          <Boton variante="texto" onClick={() => alBorrar(a.id)} disabled={ocupado}>
            Quitar
          </Boton>
        </li>
      ))}
    </ul>
  )
}
