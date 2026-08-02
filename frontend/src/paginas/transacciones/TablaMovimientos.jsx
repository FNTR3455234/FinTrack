import tabla from '../../componentes/Tabla.module.css'

import Boton from '../../componentes/Boton.jsx'
import { TipoMovimiento } from '../../componentes/Etiqueta.jsx'
import { dinero, fechaCorta } from '../../utiles/formato.js'

// Tabla de movimientos.
//
// Cuenta y categoria llegan como identificadores, no como nombres: la API
// devuelve el documento tal cual esta guardado. Los mapas que recibe este
// componente los traducen sin una peticion extra por fila.
export default function TablaMovimientos({ movimientos, cuentas, categorias, moneda, alEditar, alBorrar }) {
  return (
    <div className={tabla.envoltura}>
      <table className={tabla.tabla}>
        <thead>
          <tr>
            <th scope="col">Fecha</th>
            <th scope="col">Descripcion</th>
            <th scope="col">Categoria</th>
            <th scope="col">Cuenta</th>
            <th scope="col">Tipo</th>
            <th scope="col" className={tabla.numero}>Monto</th>
            <th scope="col"><span className="solo-lectores">Acciones</span></th>
          </tr>
        </thead>
        <tbody>
          {movimientos.map((movimiento) => {
            const categoria = categorias.get(movimiento.categoria_id)
            return (
              <tr key={movimiento.id}>
                <td className={tabla.tenue}>{fechaCorta(movimiento.fecha)}</td>

                <td className={tabla.principal}>
                  {movimiento.descripcion}
                  {/* Las notas se enseñan debajo y en gris: son contexto, no el
                      dato principal, pero esconderlas del todo obligaria a abrir
                      cada movimiento para saber si tiene algo escrito. */}
                  {movimiento.notas && (
                    <span className={tabla.tenue}> · {movimiento.notas}</span>
                  )}
                </td>

                <td>
                  {categoria && (
                    <span className={tabla.punto} style={{ background: categoria.color }} />
                  )}
                  {categoria?.nombre || '—'}
                </td>

                <td className={tabla.tenue}>{cuentas.get(movimiento.cuenta_id)?.nombre || '—'}</td>

                <td>
                  <TipoMovimiento tipo={movimiento.tipo} />
                </td>

                {/* El signo lo pone la vista, no la base: el monto siempre se
                    guarda positivo y el tipo es quien decide si suma o resta. */}
                <td
                  className={`${tabla.numero} ${
                    movimiento.tipo === 'ingreso' ? tabla.ingreso : tabla.gasto
                  }`}
                >
                  {movimiento.tipo === 'ingreso' ? '+' : '−'}
                  {dinero(movimiento.monto, moneda)}
                </td>

                <td>
                  <div className={tabla.acciones}>
                    <Boton variante="texto" onClick={() => alEditar(movimiento)}>
                      Editar
                    </Boton>
                    <Boton variante="texto" onClick={() => alBorrar(movimiento)}>
                      Borrar
                    </Boton>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
