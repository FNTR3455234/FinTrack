import { NavLink, Outlet } from 'react-router-dom'

import estilos from './Layout.module.css'

import { useAuth } from '../contexto/AuthContexto.jsx'
import { useTema } from '../contexto/TemaContexto.jsx'

// Las secciones de la app, en el orden en el que se usan: primero se mira como
// va el mes, luego se registran movimientos, y al final se administran los
// catalogos.
const SECCIONES = [
  { a: '/', texto: 'Tablero', icono: '◧' },
  { a: '/transacciones', texto: 'Movimientos', icono: '⇅' },
  { a: '/presupuestos', texto: 'Presupuestos', icono: '◎' },
  { a: '/reportes', texto: 'Reportes', icono: '◔' },
  { a: '/cuentas', texto: 'Cuentas', icono: '▤' },
  { a: '/categorias', texto: 'Categorias', icono: '◈' },
]

// Layout es el marco de todas las paginas privadas: barra lateral, cabecera y
// el hueco donde react-router pinta la pagina.
export default function Layout() {
  const { usuario, cerrarSesion } = useAuth()
  const { tema, alternar } = useTema()

  return (
    <div className={estilos.marco}>
      {/* El enlace de salto es lo primero que enfoca el tabulador: quien navega
          con teclado puede irse directo al contenido sin recorrer el menu
          entero en cada pagina. Solo se ve cuando tiene el foco. */}
      <a className={estilos.salto} href="#contenido">
        Saltar al contenido
      </a>

      <aside className={estilos.lateral}>
        <div className={estilos.marca}>
          <span className={estilos.logo} aria-hidden="true">
            F
          </span>
          <span>FinTrack</span>
        </div>

        <nav className={estilos.navegacion} aria-label="Secciones">
          {SECCIONES.map((seccion) => (
            <NavLink
              key={seccion.a}
              to={seccion.a}
              // "end" solo en la raiz: sin el, el enlace del tablero se
              // quedaria marcado como activo en todas las demas paginas.
              end={seccion.a === '/'}
              className={({ isActive }) =>
                [estilos.enlace, isActive ? estilos.activo : ''].filter(Boolean).join(' ')
              }
            >
              <span className={estilos.icono} aria-hidden="true">
                {seccion.icono}
              </span>
              {seccion.texto}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div className={estilos.columna}>
        <header className={estilos.cabecera}>
          <button
            type="button"
            className={estilos.iconoBoton}
            onClick={alternar}
            // El nombre accesible dice a que se cambia, no en que estamos: es
            // lo que el boton hace al pulsarlo.
            aria-label={tema === 'oscuro' ? 'Cambiar a tema claro' : 'Cambiar a tema oscuro'}
          >
            <span aria-hidden="true">{tema === 'oscuro' ? '☀' : '☾'}</span>
          </button>

          <NavLink to="/perfil" className={estilos.usuario}>
            {usuario?.nombre}
          </NavLink>

          <button type="button" className={estilos.salir} onClick={cerrarSesion}>
            Salir
          </button>
        </header>

        <main className={estilos.contenido} id="contenido">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
