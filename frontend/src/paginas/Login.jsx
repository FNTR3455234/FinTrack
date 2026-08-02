import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import estilos from './Autenticacion.module.css'

import Aviso from '../componentes/Aviso.jsx'
import Boton from '../componentes/Boton.jsx'
import Campo from '../componentes/Campo.jsx'
import { useAuth } from '../contexto/AuthContexto.jsx'
import { useAccion } from '../hooks/useAccion.js'

export default function Login() {
  const { iniciarSesion } = useAuth()
  const navegar = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const { ejecutar, ocupado, error } = useAccion(iniciarSesion)

  useEffect(() => {
    document.title = 'Entrar · FinTrack'
  }, [])

  async function enviar(evento) {
    evento.preventDefault()
    const resultado = await ejecutar({ email, password })
    if (resultado.ok) navegar('/', { replace: true })
  }

  return (
    <div className={estilos.pantalla}>
      <div className={estilos.caja}>
        <div className={estilos.marca}>
          <span className={estilos.logo} aria-hidden="true">
            F
          </span>
          FinTrack
        </div>
        <p className={estilos.intro}>Entra para ver como va tu mes.</p>

        {/* El mensaje viene del servidor tal cual: la API ya responde
            "El correo o la contraseña no son correctos", que no dice cual de
            los dos falla y por lo tanto no sirve para averiguar que correos
            estan registrados. */}
        {error && <Aviso tono="error">{error.texto}</Aviso>}

        <form className={estilos.formulario} onSubmit={enviar} noValidate>
          <Campo
            etiqueta="Correo"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
            required
            // autoFocus solo aqui: es el primer campo de la primera pantalla y
            // no hay nada mas que el usuario pueda querer hacer al llegar.
            autoFocus
          />
          <Campo
            etiqueta="Contraseña"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
          <Boton type="submit" ocupado={ocupado}>
            {ocupado ? 'Entrando' : 'Entrar'}
          </Boton>
        </form>

        <p className={estilos.demo}>
          Usuario de ejemplo: <code>demo@fintrack.mx</code> / <code>Demo1234!</code>
        </p>

        <p className={estilos.pie}>
          ¿Todavia no tienes cuenta? <Link to="/registro">Crea una</Link>
        </p>
      </div>
    </div>
  )
}
