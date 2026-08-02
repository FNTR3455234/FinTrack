import tabla from '../../componentes/Tabla.module.css'

import { EstadoPresupuesto } from '../../componentes/Etiqueta.jsx'
import { dinero, porcentaje } from '../../utiles/formato.js'

// Las tres tablas de la pagina de reportes. Son la version en numeros de lo que
// las graficas cuentan en formas: la misma informacion, sin depender del color.

// TablaGastos es la consulta relacional 1: transacciones cruzadas con
// categorias.
export function TablaGastos({ gastos, moneda }) {
  const total = gastos.reduce((suma, gasto) => suma + gasto.total, 0)

  return (
    <div className={tabla.envoltura}>
      <table className={tabla.tabla}>
        <thead>
          <tr>
            <th scope="col">Categoria</th>
            <th scope="col" className={tabla.numero}>Movimientos</th>
            <th scope="col" className={tabla.numero}>Total</th>
            <th scope="col" className={tabla.numero}>Peso</th>
          </tr>
        </thead>
        <tbody>
          {gastos.map((gasto) => (
            <tr key={gasto.categoria_id}>
              <td className={tabla.principal}>
                <span className={tabla.punto} style={{ background: gasto.color }} />
                <span aria-hidden="true">{gasto.icono} </span>
                {gasto.nombre}
              </td>
              <td className={`${tabla.numero} ${tabla.tenue}`}>{gasto.cantidad}</td>
              <td className={tabla.numero}>{dinero(gasto.total, moneda)}</td>
              <td className={tabla.numero}>{porcentaje(gasto.porcentaje)}</td>
            </tr>
          ))}
        </tbody>
        {/* El total va en un <tfoot> y no en una fila mas del cuerpo: no es un
            dato del mismo tipo que los otros y un lector de pantalla lo anuncia
            como lo que es. */}
        <tfoot>
          <tr>
            <th scope="row">Total</th>
            <td />
            <td className={`${tabla.numero} ${tabla.gasto}`}>{dinero(total, moneda)}</td>
            <td className={tabla.numero}>100%</td>
          </tr>
        </tfoot>
      </table>
    </div>
  )
}

// TablaPresupuestos es la consulta relacional 2: presupuestos cruzados con
// categorias y con transacciones.
export function TablaPresupuestos({ estados, moneda }) {
  return (
    <div className={tabla.envoltura}>
      <table className={tabla.tabla}>
        <thead>
          <tr>
            <th scope="col">Categoria</th>
            <th scope="col" className={tabla.numero}>Presupuestado</th>
            <th scope="col" className={tabla.numero}>Gastado</th>
            <th scope="col" className={tabla.numero}>Disponible</th>
            <th scope="col" className={tabla.numero}>Usado</th>
            <th scope="col">Estado</th>
          </tr>
        </thead>
        <tbody>
          {estados.map((estado) => (
            <tr key={estado.presupuesto_id}>
              <td className={tabla.principal}>
                <span className={tabla.punto} style={{ background: estado.color }} />
                {estado.nombre}
              </td>
              <td className={tabla.numero}>{dinero(estado.monto_limite, moneda)}</td>
              <td className={tabla.numero}>{dinero(estado.gastado, moneda)}</td>
              <td className={`${tabla.numero} ${estado.disponible < 0 ? tabla.gasto : ''}`}>
                {dinero(estado.disponible, moneda)}
              </td>
              <td className={tabla.numero}>{porcentaje(estado.porcentaje_usado)}</td>
              <td>
                <EstadoPresupuesto estado={estado.estado} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// TablaSaldos enseña de donde sale cada saldo: el inicial, lo que entro, lo que
// salio y el resultado. Asi la cifra final no es un numero que hay que creerse.
export function TablaSaldos({ saldos, moneda }) {
  return (
    <div className={tabla.envoltura}>
      <table className={tabla.tabla}>
        <thead>
          <tr>
            <th scope="col">Cuenta</th>
            <th scope="col" className={tabla.numero}>Saldo inicial</th>
            <th scope="col" className={tabla.numero}>Ingresos</th>
            <th scope="col" className={tabla.numero}>Gastos</th>
            <th scope="col" className={tabla.numero}>Saldo actual</th>
          </tr>
        </thead>
        <tbody>
          {saldos.map((saldo) => (
            <tr key={saldo.cuenta_id}>
              <td className={tabla.principal}>
                <span className={tabla.punto} style={{ background: saldo.color }} />
                {saldo.nombre}
                {saldo.archivada && <span className={tabla.tenue}> (archivada)</span>}
              </td>
              <td className={tabla.numero}>{dinero(saldo.saldo_inicial, moneda)}</td>
              <td className={`${tabla.numero} ${tabla.ingreso}`}>{dinero(saldo.ingresos, moneda)}</td>
              <td className={`${tabla.numero} ${tabla.gasto}`}>{dinero(saldo.gastos, moneda)}</td>
              <td className={`${tabla.numero} ${saldo.saldo < 0 ? tabla.gasto : tabla.ingreso}`}>
                {dinero(saldo.saldo, moneda)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
