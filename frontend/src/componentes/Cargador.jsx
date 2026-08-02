import estilos from './Cargador.module.css'

// Cargador ocupa la pantalla completa. Solo se usa en el arranque, mientras se
// comprueba si el token guardado sigue sirviendo: dentro de la app ya hay
// esqueletos, que no hacen desaparecer el resto de la interfaz.
export default function Cargador({ etiqueta = 'Cargando' }) {
  return (
    <div className={estilos.pantalla} role="status" aria-live="polite">
      <span className={estilos.rueda} aria-hidden="true" />
      <p className={estilos.texto}>{etiqueta}…</p>
    </div>
  )
}
