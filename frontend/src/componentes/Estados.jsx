import estilos from './Estados.module.css'

// Los tres estados en los que puede estar una pantalla que lee datos:
// cargando, vacia o con datos. El tercero lo pinta cada pagina; los otros dos
// viven aqui para que se vean igual en toda la app.

// Esqueleto dibuja la forma de lo que va a llegar.
//
// Se prefiere a un spinner porque no mueve la pagina: cuando llegan los datos
// ocupan el mismo sitio que ocupaba el esqueleto, en lugar de empujar todo
// hacia abajo. aria-hidden porque para un lector de pantalla no es contenido,
// solo decoracion; el aviso de que se esta cargando lo da el contenedor.
export function Esqueleto({ filas = 3, alto = 18 }) {
  return (
    <div className={estilos.esqueleto} aria-hidden="true">
      {Array.from({ length: filas }, (_, i) => (
        <div
          key={i}
          className={estilos.linea}
          // La ultima linea mas corta imita como termina un parrafo real.
          style={{ height: alto, width: i === filas - 1 ? '62%' : '100%' }}
        />
      ))}
    </div>
  )
}

// Cargando envuelve al esqueleto y anuncia el estado a los lectores de pantalla.
export function Cargando({ etiqueta = 'Cargando', filas = 3 }) {
  return (
    <div role="status" aria-live="polite">
      <span className="solo-lectores">{etiqueta}…</span>
      <Esqueleto filas={filas} />
    </div>
  )
}

// Vacio es la pantalla de "aqui todavia no hay nada".
//
// Siempre lleva una accion cuando la hay: un listado vacio sin salida deja al
// usuario sin saber que hacer, y es justo el primer momento en el que entra a
// la app.
export function Vacio({ titulo, children, accion }) {
  return (
    <div className={estilos.vacio}>
      <p className={estilos.vacioTitulo}>{titulo}</p>
      {children && <p className={estilos.vacioTexto}>{children}</p>}
      {accion && <div className={estilos.vacioAccion}>{accion}</div>}
    </div>
  )
}
