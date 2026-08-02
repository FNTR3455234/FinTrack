import { useCallback, useState } from 'react'

import estilos from './Reportes.module.css'

import { AvisoDeError } from '../componentes/Aviso.jsx'
import Encabezado from '../componentes/Encabezado.jsx'
import { Cargando, Vacio } from '../componentes/Estados.jsx'
import SelectorPeriodo from '../componentes/SelectorPeriodo.jsx'
import Tarjeta from '../componentes/Tarjeta.jsx'
import Barras from '../componentes/graficas/Barras.jsx'
import Pastel from '../componentes/graficas/Pastel.jsx'
import { TablaGastos, TablaPresupuestos, TablaSaldos } from './reportes/Tablas.jsx'
import * as reportes from '../api/reportes.js'
import { useAuth } from '../contexto/AuthContexto.jsx'
import { usePeriodo } from '../hooks/usePeriodo.js'
import { usePeticion } from '../hooks/usePeticion.js'
import { periodoLargo } from '../utiles/formato.js'

// Cuantos meses puede pedir la grafica de tendencia. El maximo de la API son 24.
const OPCIONES_MESES = [6, 12, 24]

export default function Reportes() {
  const { usuario } = useAuth()
  const { periodo, mover, alMesActual, esMesActual } = usePeriodo()
  const [meses, setMeses] = useState(OPCIONES_MESES[0])

  const cargar = useCallback(
    ({ signal }) =>
      Promise.all([
        reportes.gastosPorCategoria(periodo, { signal }),
        reportes.estadoPresupuestos(periodo, { signal }),
        reportes.tendencia({ ...periodo, meses }, { signal }),
        reportes.saldos(undefined, { signal }),
      ]).then(([gastos, presupuestos, tendencia, saldos]) => ({
        gastos,
        presupuestos,
        tendencia,
        saldos,
      })),
    [periodo, meses],
  )

  const { datos, cargando, error, recargar } = usePeticion(cargar, [periodo.mes, periodo.anio, meses])

  return (
    <>
      <Encabezado titulo="Reportes" descripcion={`Analisis de ${periodoLargo(periodo)}.`}>
        <SelectorPeriodo
          periodo={periodo}
          mover={mover}
          alMesActual={alMesActual}
          esMesActual={esMesActual}
        />
      </Encabezado>

      {error && <AvisoDeError error={error} alReintentar={recargar} />}
      {!datos && cargando && <Cargando etiqueta="Cargando los reportes" filas={6} />}

      {datos && (
        <div className={estilos.pila} data-cargando={cargando || undefined}>
          <Tarjeta
            titulo="Gastos por categoria"
            subtitulo="Cruza transacciones con categorias: cuanto y en que se fue el dinero del mes."
          >
            {datos.gastos.length === 0 ? (
              <Vacio titulo="Sin gastos en este mes" />
            ) : (
              <div className={estilos.dosColumnas}>
                <div className={estilos.grafica}>
                  <Pastel gastos={datos.gastos} moneda={usuario.moneda} conTabla={false} />
                </div>
                <TablaGastos gastos={datos.gastos} moneda={usuario.moneda} />
              </div>
            )}
          </Tarjeta>

          <Tarjeta
            titulo="Estado de los presupuestos"
            subtitulo="Cruza presupuestos con categorias y transacciones: lo planeado contra lo real."
          >
            {datos.presupuestos.length === 0 ? (
              <Vacio titulo={`Sin presupuestos en ${periodoLargo(periodo)}`} />
            ) : (
              <TablaPresupuestos estados={datos.presupuestos} moneda={usuario.moneda} />
            )}
          </Tarjeta>

          <Tarjeta
            titulo="Tendencia"
            subtitulo="Ingresos contra gastos, mes a mes."
            accion={
              <label className={estilos.selectorMeses}>
                <span className="solo-lectores">Cuantos meses mostrar</span>
                <select value={meses} onChange={(evento) => setMeses(Number(evento.target.value))}>
                  {OPCIONES_MESES.map((opcion) => (
                    <option key={opcion} value={opcion}>
                      Ultimos {opcion} meses
                    </option>
                  ))}
                </select>
              </label>
            }
          >
            <Barras serie={datos.tendencia} moneda={usuario.moneda} />
          </Tarjeta>

          <Tarjeta
            titulo="Saldo de cada cuenta"
            subtitulo="saldo inicial + ingresos − gastos. No se guarda: se calcula al preguntarlo."
          >
            {datos.saldos.length === 0 ? (
              <Vacio titulo="Todavia no tienes cuentas" />
            ) : (
              <TablaSaldos saldos={datos.saldos} moneda={usuario.moneda} />
            )}
          </Tarjeta>
        </div>
      )}
    </>
  )
}
