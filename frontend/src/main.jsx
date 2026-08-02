import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'

import App from './App.jsx'
import { ProveedorAuth } from './contexto/AuthContexto.jsx'
import { ProveedorTema } from './contexto/TemaContexto.jsx'
import './estilos/global.css'

// Punto de entrada. El orden de los proveedores importa: el tema envuelve todo
// para que hasta la pantalla de login se pinte con el tema elegido, y la
// autenticacion va dentro del router porque necesita redirigir.
createRoot(document.getElementById('raiz')).render(
  <StrictMode>
    <ProveedorTema>
      <BrowserRouter>
        <ProveedorAuth>
          <App />
        </ProveedorAuth>
      </BrowserRouter>
    </ProveedorTema>
  </StrictMode>,
)
