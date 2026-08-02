import estilos from './Etiqueta.module.css'

// Textos del semaforo de presupuestos. Los codigos (ok, alerta, excedido) los
// decide la API; aqui solo se traducen a algo que se pueda leer.
const TEXTO_ESTADO = {
  ok: 'En orden',
  alerta: 'Cerca del limite',
  excedido: 'Excedido',
}

// Etiqueta es la pastilla de color de las tablas.
//
// Nunca lleva solo color: el estado tambien va escrito. Un semaforo que solo se
// distingue por el tono deja fuera a quien no distingue el rojo del verde, que
// es alrededor de uno de cada doce hombres.
export default function Etiqueta({ tono = 'neutro', children }) {
  return (
    <span className={estilos.etiqueta} data-tono={tono}>
      {children}
    </span>
  )
}

// EstadoPresupuesto pinta el semaforo a partir del codigo que da la API.
export function EstadoPresupuesto({ estado }) {
  return <Etiqueta tono={estado}>{TEXTO_ESTADO[estado] || estado}</Etiqueta>
}

// TipoMovimiento distingue ingreso de gasto en el listado de transacciones.
export function TipoMovimiento({ tipo }) {
  return <Etiqueta tono={tipo}>{tipo === 'ingreso' ? 'Ingreso' : 'Gasto'}</Etiqueta>
}
