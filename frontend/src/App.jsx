import { Suspense, lazy } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'

import Layout from './componentes/Layout.jsx'
import Cargador from './componentes/Cargador.jsx'
import { useAuth } from './contexto/AuthContexto.jsx'
import Categorias from './paginas/Categorias.jsx'
import Cuentas from './paginas/Cuentas.jsx'
import Login from './paginas/Login.jsx'
import NoEncontrada from './paginas/NoEncontrada.jsx'
import Perfil from './paginas/Perfil.jsx'
import Presupuestos from './paginas/Presupuestos.jsx'
import Registro from './paginas/Registro.jsx'
import Transacciones from './paginas/Transacciones.jsx'

// Las dos paginas con graficas se cargan aparte.
//
// Recharts pesa mas que todo el resto de la aplicacion junta. Con lazy(), quien
// entra al login o registra un movimiento no descarga esa libreria: solo baja
// cuando de verdad se abre una pantalla con graficas. Son las unicas dos rutas
// que se parten, porque son las unicas que lo justifican.
const Tablero = lazy(() => import('./paginas/Tablero.jsx'))
const Reportes = lazy(() => import('./paginas/Reportes.jsx'))

// Mapa de la aplicacion.
//
// Las rutas privadas cuelgan de un mismo <Route> con el Layout: asi la barra
// lateral no se vuelve a montar al navegar entre paginas, y la comprobacion de
// sesion se escribe una sola vez en lugar de en cada pagina.
export default function App() {
  const { cargando } = useAuth()

  // Mientras se comprueba el token guardado no se decide nada: pintar el login
  // aqui haria parpadear la pantalla de entrada a quien ya tiene sesion.
  if (cargando) return <Cargador etiqueta="Comprobando la sesion" />

  return (
    // Suspense es lo que se ve mientras baja el trozo de una ruta perezosa.
    <Suspense fallback={<Cargador etiqueta="Cargando la seccion" />}>
      <Routes>
        <Route path="/login" element={<SoloInvitados><Login /></SoloInvitados>} />
        <Route path="/registro" element={<SoloInvitados><Registro /></SoloInvitados>} />

        <Route element={<RutaPrivada />}>
          <Route path="/" element={<Tablero />} />
          <Route path="/transacciones" element={<Transacciones />} />
          <Route path="/cuentas" element={<Cuentas />} />
          <Route path="/categorias" element={<Categorias />} />
          <Route path="/presupuestos" element={<Presupuestos />} />
          <Route path="/reportes" element={<Reportes />} />
          <Route path="/perfil" element={<Perfil />} />
        </Route>

        <Route path="*" element={<NoEncontrada />} />
      </Routes>
    </Suspense>
  )
}

// RutaPrivada manda al login a quien no tiene sesion y, si la tiene, pinta el
// Layout con la pagina dentro.
function RutaPrivada() {
  const { usuario } = useAuth()
  if (!usuario) return <Navigate to="/login" replace />
  return <Layout />
}

// SoloInvitados evita que alguien con la sesion abierta se quede mirando el
// formulario de login si escribe la direccion a mano.
function SoloInvitados({ children }) {
  const { usuario } = useAuth()
  if (usuario) return <Navigate to="/" replace />
  return children
}
