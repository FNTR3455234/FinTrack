import estilos from './Tarjeta.module.css'

// Tarjeta es el panel blanco (o gris oscuro) sobre el que va todo: tablas,
// graficas, formularios. Tener un solo componente para eso evita que cada
// pantalla invente su propio borde y su propia sombra.
export default function Tarjeta({ titulo, subtitulo, accion, children, className = '' }) {
  return (
    <section className={[estilos.tarjeta, className].filter(Boolean).join(' ')}>
      {(titulo || accion) && (
        <header className={estilos.cabecera}>
          <div>
            {titulo && <h2 className={estilos.titulo}>{titulo}</h2>}
            {subtitulo && <p className={estilos.subtitulo}>{subtitulo}</p>}
          </div>
          {accion}
        </header>
      )}
      <div className={estilos.cuerpo}>{children}</div>
    </section>
  )
}

// Cifra es la tarjeta pequeña del tablero: un rotulo, un numero grande y un pie
// opcional. El tono pinta el numero de verde o rojo sin repetir estilos en cada
// pantalla.
export function Cifra({ rotulo, valor, pie, tono = 'neutro' }) {
  return (
    <div className={estilos.cifra}>
      <p className={estilos.rotulo}>{rotulo}</p>
      {/*
        Sin la clase .dinero a proposito: las cifras tabulares alinean columnas
        de numeros, pero a este tamaño hacen que "121" se vea suelto. Aqui el
        numero va solo, no en una columna.
      */}
      <p className={estilos.valor} data-tono={tono}>
        {valor}
      </p>
      {pie && <p className={estilos.pieCifra}>{pie}</p>}
    </div>
  )
}
