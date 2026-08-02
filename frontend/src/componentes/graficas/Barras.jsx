import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

import estilos from './Graficas.module.css'
import { EJE, Globo, Leyenda, REJILLA, TablaDeDatos } from './comun.jsx'

import { dinero, dineroCorto, nombreMes } from '../../utiles/formato.js'

// Las dos series de la grafica de tendencia. El orden es fijo: los ingresos
// siempre a la izquierda de cada par y los gastos a la derecha.
//
// Esa posicion fija es la segunda pista de identidad, ademas de la leyenda y el
// globo. Es lo que hace que la grafica se siga leyendo aunque los dos colores
// se parezcan (por ejemplo, al imprimirla en blanco y negro).
const SERIES = [
  { clave: 'ingresos', nombre: 'Ingresos', color: 'var(--serie-ingresos)' },
  { clave: 'gastos', nombre: 'Gastos', color: 'var(--serie-gastos)' },
]

// Barras compara ingresos contra gastos mes a mes.
//
// Un solo eje vertical para las dos series. Poner cada una en su propio eje
// haria que las alturas se pudieran ajustar a voluntad y la grafica insinuaria
// una relacion entre ambas que no esta en los datos.
export default function Barras({ serie, moneda }) {
  const datos = serie.map((punto) => ({ ...punto, etiqueta: etiquetaCorta(punto.periodo) }))

  return (
    <>
      <div className={estilos.contenedor}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={datos} margin={{ top: 8, right: 4, bottom: 0, left: -12 }} barGap={2}>
            <CartesianGrid {...REJILLA} />
            <XAxis dataKey="etiqueta" {...EJE} />
            <YAxis tickFormatter={dineroCorto} width={56} {...EJE} />

            <Tooltip
              // El resaltado de Recharts por defecto es un bloque gris opaco que
              // tapa las barras; se cambia por un tono muy suave del texto.
              cursor={{ fill: 'var(--texto-tenue)', fillOpacity: 0.08 }}
              content={({ active, payload, label }) =>
                active && payload?.length ? (
                  <Globo
                    titulo={label}
                    moneda={moneda}
                    filas={SERIES.map((s) => ({
                      nombre: s.nombre,
                      color: s.color,
                      valor: payload.find((p) => p.dataKey === s.clave)?.value ?? 0,
                    }))}
                  />
                ) : null
              }
            />

            {SERIES.map((s) => (
              <Bar
                key={s.clave}
                dataKey={s.clave}
                fill={s.color}
                // Barras finas con la punta redondeada y la base cuadrada, y un
                // hueco de 2px entre las dos del mismo mes.
                maxBarSize={22}
                radius={[4, 4, 0, 0]}
                isAnimationActive={false}
              />
            ))}
          </BarChart>
        </ResponsiveContainer>
      </div>

      <Leyenda claves={SERIES.map((s) => ({ nombre: s.nombre, color: s.color }))} />

      <TablaDeDatos
        resumen="Ver los datos en una tabla"
        columnas={['Mes', 'Ingresos', 'Gastos', 'Balance']}
        filas={datos.map((punto) => [
          punto.etiqueta,
          dinero(punto.ingresos, moneda),
          dinero(punto.gastos, moneda),
          dinero(punto.balance, moneda),
        ])}
      />
    </>
  )
}

// etiquetaCorta convierte { mes: 7, anio: 2026 } en "jul 26". La API tambien
// manda una etiqueta ("2026-07"), pero en el eje se lee peor.
function etiquetaCorta({ mes, anio }) {
  return `${nombreMes(mes).slice(0, 3)} ${String(anio).slice(2)}`
}
