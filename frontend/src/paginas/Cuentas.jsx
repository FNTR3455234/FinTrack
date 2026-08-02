import { useCallback, useState } from 'react'

import tabla from '../componentes/Tabla.module.css'

import { AvisoDeError } from '../componentes/Aviso.jsx'
import Boton from '../componentes/Boton.jsx'
import ConfirmarBorrado from '../componentes/ConfirmarBorrado.jsx'
import Encabezado from '../componentes/Encabezado.jsx'
import { Cargando, Vacio } from '../componentes/Estados.jsx'
import Etiqueta from '../componentes/Etiqueta.jsx'
import Tarjeta from '../componentes/Tarjeta.jsx'
import FormularioCuenta from './cuentas/FormularioCuenta.jsx'
import { cuentas as api } from '../api/catalogos.js'
import * as reportes from '../api/reportes.js'
import { useAuth } from '../contexto/AuthContexto.jsx'
import { usePeticion } from '../hooks/usePeticion.js'
import { dinero } from '../utiles/formato.js'

// La lista se pide a /reportes/saldos y no a /cuentas.
//
// Ese reporte devuelve las mismas cuentas con todos sus campos MAS el saldo de
// hoy, que es el dato por el que se entra a esta pantalla. Pedir las dos cosas
// por separado seria una peticion mas para acabar en lo mismo.
export default function Cuentas() {
  const { usuario } = useAuth()
  const [editando, setEditando] = useState(null)
  const [borrando, setBorrando] = useState(null)

  const cargar = useCallback(({ signal }) => reportes.saldos(undefined, { signal }), [])
  const { datos, cargando, error, recargar } = usePeticion(cargar)

  async function guardar(valores) {
    if (editando.cuenta_id) {
      await api.actualizar(editando.cuenta_id, valores)
    } else {
      await api.crear(valores)
    }
    recargar()
  }

  return (
    <>
      <Encabezado titulo="Cuentas" descripcion="De donde sale y a donde entra tu dinero.">
        <Boton onClick={() => setEditando({})}>Nueva cuenta</Boton>
      </Encabezado>

      {error && <AvisoDeError error={error} alReintentar={recargar} />}

      <Tarjeta>
        {cargando && !datos && <Cargando etiqueta="Cargando las cuentas" filas={4} />}

        {datos && datos.length === 0 && (
          <Vacio
            titulo="Todavia no tienes cuentas"
            accion={<Boton onClick={() => setEditando({})}>Crear la primera</Boton>}
          >
            Una cuenta es donde guardas el dinero: tu efectivo, tu cuenta del banco o una
            tarjeta.
          </Vacio>
        )}

        {datos && datos.length > 0 && (
          <div className={tabla.envoltura}>
            <table className={tabla.tabla}>
              <thead>
                <tr>
                  <th scope="col">Cuenta</th>
                  <th scope="col">Tipo</th>
                  <th scope="col" className={tabla.numero}>Saldo inicial</th>
                  <th scope="col" className={tabla.numero}>Movimientos</th>
                  <th scope="col" className={tabla.numero}>Saldo actual</th>
                  <th scope="col"><span className="solo-lectores">Acciones</span></th>
                </tr>
              </thead>
              <tbody>
                {datos.map((cuenta) => (
                  <tr key={cuenta.cuenta_id}>
                    <td className={tabla.principal}>
                      <span className={tabla.punto} style={{ background: cuenta.color }} />
                      {cuenta.nombre}
                      {cuenta.archivada && ' '}
                      {cuenta.archivada && <Etiqueta>Archivada</Etiqueta>}
                    </td>
                    <td className={tabla.tenue}>{cuenta.tipo}</td>
                    <td className={tabla.numero}>{dinero(cuenta.saldo_inicial, usuario.moneda)}</td>
                    <td className={`${tabla.numero} ${tabla.tenue}`}>
                      +{dinero(cuenta.ingresos, usuario.moneda)} / −
                      {dinero(cuenta.gastos, usuario.moneda)}
                    </td>
                    <td
                      className={`${tabla.numero} ${cuenta.saldo < 0 ? tabla.gasto : tabla.ingreso}`}
                    >
                      {dinero(cuenta.saldo, usuario.moneda)}
                    </td>
                    <td>
                      <div className={tabla.acciones}>
                        <Boton variante="texto" onClick={() => setEditando(cuenta)}>
                          Editar
                        </Boton>
                        <Boton variante="texto" onClick={() => setBorrando(cuenta)}>
                          Borrar
                        </Boton>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Tarjeta>

      {/* La clave hace que el formulario se monte de cero para cada cuenta: sin
          ella React reutilizaria el mismo componente y conservaria los valores
          de la cuenta anterior. */}
      {editando && (
        <FormularioCuenta
          key={editando.cuenta_id || 'nueva'}
          abierto
          cuenta={editando.cuenta_id ? editando : null}
          alGuardar={guardar}
          alCerrar={() => setEditando(null)}
        />
      )}

      {borrando && (
        <ConfirmarBorrado
          abierto
          nombre={borrando.nombre}
          alCerrar={() => setBorrando(null)}
          alConfirmar={() => api.eliminar(borrando.cuenta_id).then(recargar)}
        />
      )}
    </>
  )
}
