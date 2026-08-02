import { useCallback } from 'react'
import { Link } from 'react-router-dom'

import estilos from './Tablero.module.css'

import { AvisoDeError } from '../componentes/Aviso.jsx'
import BarraProgreso from '../componentes/BarraProgreso.jsx'
import Encabezado from '../componentes/Encabezado.jsx'
import { Cargando, Vacio } from '../componentes/Estados.jsx'
import SelectorPeriodo from '../componentes/SelectorPeriodo.jsx'
import Tarjeta, { Cifra } from '../componentes/Tarjeta.jsx'
import Barras from '../componentes/graficas/Barras.jsx'
import Pastel from '../componentes/graficas/Pastel.jsx'
import { useAuth } from '../contexto/AuthContexto.jsx'
import { usePeriodo } from '../hooks/usePeriodo.js'
import { usePeticion } from '../hooks/usePeticion.js'
import * as reportes from '../api/reportes.js'
import { dinero, periodoLargo } from '../utiles/formato.js'

// Cuantos meses de historia se pintan en la grafica de tendencia.
const MESES_TENDENCIA = 6

export default function Tablero() {
  const { usuario } = useAuth()
  const { periodo, mover, alMesActual, esMesActual } = usePeriodo()

  // Las cuatro consultas del tablero salen juntas y se esperan juntas. Cuatro
  // usePeticion separados harian que la pagina se armara a trozos, con cada
  // tarjeta apareciendo cuando le tocara.
  const cargar = useCallback(
    ({ signal }) =>
      Promise.all([
        reportes.resumen(periodo, { signal }),
        reportes.gastosPorCategoria(periodo, { signal }),
        reportes.estadoPresupuestos(periodo, { signal }),
        reportes.tendencia({ ...periodo, meses: MESES_TENDENCIA }, { signal }),
      ]).then(([resumen, gastos, presupuestos, tendencia]) => ({
        resumen,
        gastos,
        presupuestos,
        tendencia,
      })),
    [periodo],
  )

  const { datos, cargando, error, recargar } = usePeticion(cargar, [periodo.mes, periodo.anio])

  return (
    <>
      <Encabezado titulo="Tablero" descripcion={`Como va ${periodoLargo(periodo)}.`}>
        {/* Un solo control de periodo arriba, y todas las tarjetas responden a
            el. Un filtro por tarjeta dejaria la pantalla mostrando meses
            distintos a la vez. */}
        <SelectorPeriodo
          periodo={periodo}
          mover={mover}
          alMesActual={alMesActual}
          esMesActual={esMesActual}
        />
      </Encabezado>

      {error && <AvisoDeError error={error} alReintentar={recargar} />}

      {/* Mientras llegan los datos nuevos se conserva la pantalla anterior a
          media opacidad. Volver al esqueleto en cada cambio de mes haria saltar
          el contenido y perderia el sitio en el que estaba el usuario. */}
      {!datos && cargando && <Cargando etiqueta="Cargando el tablero" filas={5} />}

      {datos && (
        <div className={estilos.tablero} data-cargando={cargando || undefined}>
          <Resumen resumen={datos.resumen} moneda={usuario.moneda} />

          <div className={estilos.columnas}>
            <Tarjeta
              titulo="En que se fue el dinero"
              subtitulo="Gasto del mes por categoria"
              className={estilos.ancha}
            >
              {datos.gastos.length === 0 ? (
                <Vacio titulo="Sin gastos en este mes">
                  Cuando registres un gasto, aqui veras el reparto por categoria.
                </Vacio>
              ) : (
                <Pastel gastos={datos.gastos} moneda={usuario.moneda} />
              )}
            </Tarjeta>

            <Tarjeta
              titulo="Ultimos meses"
              subtitulo={`Ingresos contra gastos, ${MESES_TENDENCIA} meses`}
              className={estilos.ancha}
            >
              <Barras serie={datos.tendencia} moneda={usuario.moneda} />
            </Tarjeta>
          </div>

          <Presupuestos estados={datos.presupuestos} />
        </div>
      )}
    </>
  )
}

// Resumen son las cuatro cifras de arriba.
function Resumen({ resumen, moneda }) {
  return (
    <div className={estilos.cifras}>
      <Cifra rotulo="Ingresos" valor={dinero(resumen.ingresos, moneda)} tono="ingreso" />
      <Cifra rotulo="Gastos" valor={dinero(resumen.gastos, moneda)} tono="gasto" />
      <Cifra
        rotulo="Balance"
        valor={dinero(resumen.balance, moneda)}
        tono={resumen.balance < 0 ? 'gasto' : 'ingreso'}
        pie={`${resumen.movimientos} movimientos`}
      />
      <Cifra
        rotulo="Saldo disponible"
        valor={dinero(resumen.saldo_total, moneda)}
        pie="Suma de tus cuentas activas"
      />
    </div>
  )
}

// Presupuestos pinta una barra por cada presupuesto del mes.
function Presupuestos({ estados }) {
  return (
    <Tarjeta
      titulo="Presupuestos del mes"
      accion={<Link to="/presupuestos">Administrar</Link>}
    >
      {estados.length === 0 ? (
        <Vacio
          titulo="Todavia no tienes presupuestos"
          accion={<Link to="/presupuestos">Crear el primero</Link>}
        >
          Ponle un limite mensual a una categoria de gasto y aqui veras cuanto llevas.
        </Vacio>
      ) : (
        <div className={estilos.barras}>
          {estados.map((estado) => (
            <BarraProgreso key={estado.presupuesto_id} estado={estado} />
          ))}
        </div>
      )}
    </Tarjeta>
  )
}
