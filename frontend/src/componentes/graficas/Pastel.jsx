import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts'

import estilos from './Graficas.module.css'
import { Globo, Leyenda, TablaDeDatos } from './comun.jsx'

import { dinero, porcentaje } from '../../utiles/formato.js'

// Cuantas porciones se dibujan antes de agrupar el resto en "Otras".
//
// Un pastel deja de leerse pasadas seis porciones: las de abajo se vuelven
// astillas que no se pueden comparar entre si. La cola no se pierde, se suma en
// una sola porcion gris y sigue entera en la tabla de datos.
const MAXIMO_PORCIONES = 6
const COLOR_OTRAS = 'var(--texto-tenue)'

// Pastel muestra el peso de cada categoria en el gasto del mes (la consulta
// relacional 1).
//
// Los colores salen de la base: son los que el usuario le puso a cada
// categoria. Por eso no se pueden validar de antemano, y por eso la grafica
// nunca depende solo del color: cada porcion tiene su nombre en la leyenda, su
// monto en el globo y su fila en la tabla.
// conTabla se apaga en la pagina de reportes, donde al lado ya esta la tabla
// completa: repetir los mismos numeros dos veces en la misma tarjeta no añade
// nada y solo estorba.
export default function Pastel({ gastos, moneda, conTabla = true }) {
  const porciones = agrupar(gastos)

  return (
    <>
      {/*
        role="img" con su etiqueta: para un lector de pantalla esto es UNA
        imagen con nombre, no un monton de <path> sueltos. Recharts marca cada
        porcion como imagen sin texto alternativo, y sin esto se anuncian seis
        "imagen" seguidas que no dicen nada.

        Los numeros no van en la etiqueta: estan enteros en la tabla de abajo,
        que es donde se pueden leer uno por uno.
      */}
      <div
        className={estilos.contenedor}
        role="img"
        aria-label="Grafica de reparto del gasto por categoria. Los mismos datos estan en la tabla."
      >
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={porciones}
              dataKey="total"
              nameKey="nombre"
              innerRadius="52%"
              outerRadius="80%"
              // El hueco de 2px en el color de la superficie separa las
              // porciones. Es un hueco, no un borde: un trazo alrededor de cada
              // marca añade tinta que no es dato.
              stroke="var(--superficie)"
              strokeWidth={2}
              // Sin animacion de entrada: la grafica se vuelve a montar cada vez
              // que cambia el mes y el giro constante distrae mas que informa.
              isAnimationActive={false}
            >
              {porciones.map((porcion) => (
                // aria-hidden en cada porcion: Recharts las marca una por una
                // como role="img" sin texto alternativo, y un lector de
                // pantalla acabaria anunciando seis "imagen" seguidas. El
                // nombre lo pone el contenedor de aqui arriba, una sola vez.
                <Cell key={porcion.nombre} fill={porcion.color} aria-hidden="true" />
              ))}
            </Pie>

            <Tooltip
              content={({ active, payload }) =>
                active && payload?.length ? (
                  <Globo
                    titulo={payload[0].payload.nombre}
                    moneda={moneda}
                    filas={[
                      {
                        nombre: porcentaje(payload[0].payload.porcentaje),
                        valor: payload[0].payload.total,
                        color: payload[0].payload.color,
                      },
                    ]}
                  />
                ) : null
              }
            />
          </PieChart>
        </ResponsiveContainer>
      </div>

      <Leyenda claves={porciones.map((p) => ({ nombre: p.nombre, color: p.color }))} />

      {conTabla && (
        <TablaDeDatos
          resumen="Ver los datos en una tabla"
          columnas={['Categoria', 'Total', 'Peso']}
          filas={gastos.map((g) => [g.nombre, dinero(g.total, moneda), porcentaje(g.porcentaje)])}
        />
      )}
    </>
  )
}

// agrupar deja las categorias mas grandes y suma el resto en una porcion
// "Otras". La API ya devuelve la lista ordenada de mayor a menor.
function agrupar(gastos) {
  if (gastos.length <= MAXIMO_PORCIONES) return gastos

  const cabeza = gastos.slice(0, MAXIMO_PORCIONES - 1)
  const cola = gastos.slice(MAXIMO_PORCIONES - 1)

  return [
    ...cabeza,
    {
      nombre: `Otras (${cola.length})`,
      color: COLOR_OTRAS,
      total: cola.reduce((suma, g) => suma + g.total, 0),
      porcentaje: cola.reduce((suma, g) => suma + g.porcentaje, 0),
    },
  ]
}
