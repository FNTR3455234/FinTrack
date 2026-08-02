import estilos from './Filtros.module.css'

import Campo from '../../componentes/Campo.jsx'

// Los cuatro ordenamientos que acepta la API.
const ORDENES = [
  { valor: 'fecha_desc', texto: 'Mas recientes primero' },
  { valor: 'fecha_asc', texto: 'Mas antiguos primero' },
  { valor: 'monto_desc', texto: 'Monto de mayor a menor' },
  { valor: 'monto_asc', texto: 'Monto de menor a mayor' },
]

// Filtros es una sola fila encima de la tabla, no un filtro por columna.
//
// Todos los controles escriben en el mismo objeto de estado y ese objeto es el
// que viaja como query a la API: lo que se ve en pantalla y lo que se le pide al
// servidor no pueden separarse.
export default function Filtros({ valores, alCambiar, cuentas, categorias, alLimpiar }) {
  function cambiar(campo) {
    return (evento) => alCambiar(campo, evento.target.value)
  }

  const hayFiltros = Object.entries(valores).some(
    ([campo, valor]) => valor !== '' && campo !== 'orden',
  )

  return (
    <div className={estilos.filtros}>
      <Campo etiqueta="Buscar" type="search" value={valores.busqueda} onChange={cambiar('busqueda')} />

      <Campo etiqueta="Desde" type="date" value={valores.desde} onChange={cambiar('desde')} />
      <Campo etiqueta="Hasta" type="date" value={valores.hasta} onChange={cambiar('hasta')} />

      <Campo etiqueta="Tipo">
        <select value={valores.tipo} onChange={cambiar('tipo')}>
          <option value="">Todos</option>
          <option value="ingreso">Ingresos</option>
          <option value="gasto">Gastos</option>
        </select>
      </Campo>

      <Campo etiqueta="Cuenta">
        <select value={valores.cuenta_id} onChange={cambiar('cuenta_id')}>
          <option value="">Todas</option>
          {cuentas.map((cuenta) => (
            <option key={cuenta.id} value={cuenta.id}>
              {cuenta.nombre}
            </option>
          ))}
        </select>
      </Campo>

      <Campo etiqueta="Categoria">
        <select value={valores.categoria_id} onChange={cambiar('categoria_id')}>
          <option value="">Todas</option>
          {categorias.map((categoria) => (
            <option key={categoria.id} value={categoria.id}>
              {categoria.nombre}
            </option>
          ))}
        </select>
      </Campo>

      <Campo etiqueta="Orden">
        <select value={valores.orden} onChange={cambiar('orden')}>
          {ORDENES.map((orden) => (
            <option key={orden.valor} value={orden.valor}>
              {orden.texto}
            </option>
          ))}
        </select>
      </Campo>

      {/* El boton de limpiar solo aparece cuando hay algo que limpiar: un boton
          que no hace nada enseña al usuario a no fiarse de los botones. */}
      {hayFiltros && (
        <button type="button" className={estilos.limpiar} onClick={alLimpiar}>
          Quitar filtros
        </button>
      )}
    </div>
  )
}
