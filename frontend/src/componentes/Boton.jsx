import estilos from './Boton.module.css'

// Boton unico de la app, con tres variantes.
//
// Existe sobre todo por dos detalles que es facil olvidar en cada formulario:
// type="button" por defecto (si no, cualquier boton dentro de un <form> lo
// envia) y el bloqueo mientras hay una peticion en curso, que es lo que evita
// crear la misma transaccion dos veces por doble clic.
export default function Boton({
  variante = 'primario',
  ocupado = false,
  disabled = false,
  type = 'button',
  className = '',
  children,
  ...resto
}) {
  const clases = [estilos.boton, estilos[variante], className].filter(Boolean).join(' ')

  return (
    <button
      type={type}
      className={clases}
      disabled={disabled || ocupado}
      // aria-busy le dice al lector de pantalla que la accion sigue en marcha,
      // que es la parte que el spinner solo comunica a quien ve la pantalla.
      aria-busy={ocupado || undefined}
      {...resto}
    >
      {ocupado && <span className={estilos.rueda} aria-hidden="true" />}
      {children}
    </button>
  )
}
