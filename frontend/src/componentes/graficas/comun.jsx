import estilos from './Graficas.module.css'

import { dinero } from '../../utiles/formato.js'

// Piezas que comparten las graficas.
//
// Los colores se escriben como var(--...) directamente en los atributos del SVG.
// Recharts los pasa tal cual y el navegador los resuelve, asi que al cambiar de
// tema las graficas se repintan solas: no hay que leer los estilos calculados
// desde JavaScript ni volver a montar nada.
export const EJE = {
  stroke: 'var(--texto-tenue)',
  fontSize: 11,
  tickLine: false,
  axisLine: false,
}

// La rejilla es una linea de un solo pixel, continua y del color del borde.
// Nada de rayas: una rejilla punteada se lee como "umbral" o "proyeccion"
// cuando solo es la rejilla.
export const REJILLA = {
  stroke: 'var(--borde)',
  strokeDasharray: '',
  vertical: false,
}

// Globo compartido por las dos graficas. Recibe las series ya listas para
// pintar: { nombre, valor, color }.
export function Globo({ titulo, filas, moneda }) {
  return (
    <div className={estilos.globo}>
      <p className={estilos.globoTitulo}>{titulo}</p>
      {filas.map((fila) => (
        <div key={fila.nombre} className={estilos.globoFila}>
          <span className={estilos.claveLeyenda}>
            <span className={estilos.muestra} style={{ background: fila.color }} />
            {fila.nombre}
          </span>
          <span className={estilos.globoValor}>{dinero(fila.valor, moneda)}</span>
        </div>
      ))}
    </div>
  )
}

// Leyenda propia en vez de la de Recharts: asi el texto usa el color de texto
// del tema y no el de la serie, y la muestra de color queda al lado en lugar de
// dentro de la palabra.
export function Leyenda({ claves }) {
  return (
    <div className={estilos.leyenda}>
      {claves.map((clave) => (
        <span key={clave.nombre} className={estilos.claveLeyenda}>
          <span className={estilos.muestra} style={{ background: clave.color }} />
          {clave.nombre}
        </span>
      ))}
    </div>
  )
}

// TablaDeDatos es el gemelo accesible de cada grafica: los mismos numeros sin
// depender del color. Un tooltip no puede ser la unica forma de leer un valor.
export function TablaDeDatos({ resumen, columnas, filas }) {
  return (
    <details className={estilos.detalle}>
      <summary>{resumen}</summary>
      <table className={estilos.tablaDatos}>
        <thead>
          <tr>
            {columnas.map((columna) => (
              <th key={columna} scope="col">
                {columna}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {filas.map((fila) => (
            <tr key={fila[0]}>
              {fila.map((celda, indice) => (
                <td key={indice}>{celda}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </details>
  )
}
