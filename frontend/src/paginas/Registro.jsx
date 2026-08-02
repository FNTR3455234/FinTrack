import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import estilos from './Autenticacion.module.css'

import Aviso from '../componentes/Aviso.jsx'
import Boton from '../componentes/Boton.jsx'
import Campo from '../componentes/Campo.jsx'
import { useAuth } from '../contexto/AuthContexto.jsx'
import { useAccion } from '../hooks/useAccion.js'

// Las reglas del formulario son las mismas que valida el servidor (ver las
// etiquetas binding de modelos.PeticionRegistro). Comprobarlas aqui tambien no
// es duplicar por gusto: ahorra un viaje al servidor y da el error debajo del
// campo que lo causa, no en un aviso general arriba.
const MINIMO_PASSWORD = 8

export default function Registro() {
  const { registrarse } = useAuth()
  const navegar = useNavigate()
  const [datos, setDatos] = useState({ nombre: '', email: '', password: '' })
  const [tocado, setTocado] = useState(false)
  const { ejecutar, ocupado, error } = useAccion(registrarse)

  useEffect(() => {
    document.title = 'Crear cuenta · FinTrack'
  }, [])

  const errorPassword =
    tocado && datos.password.length < MINIMO_PASSWORD
      ? `La contraseña necesita al menos ${MINIMO_PASSWORD} caracteres.`
      : null

  function cambiar(campo) {
    return (evento) => setDatos((actual) => ({ ...actual, [campo]: evento.target.value }))
  }

  async function enviar(evento) {
    evento.preventDefault()
    setTocado(true)
    if (datos.password.length < MINIMO_PASSWORD) return

    const resultado = await ejecutar(datos)
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
        <p className={estilos.intro}>Crea tu cuenta y empieza a registrar tus movimientos.</p>

        {error && <Aviso tono="error">{error.texto}</Aviso>}

        <form className={estilos.formulario} onSubmit={enviar} noValidate>
          <Campo
            etiqueta="Nombre"
            value={datos.nombre}
            onChange={cambiar('nombre')}
            autoComplete="name"
            minLength={2}
            maxLength={80}
            required
            autoFocus
          />
          <Campo
            etiqueta="Correo"
            type="email"
            value={datos.email}
            onChange={cambiar('email')}
            autoComplete="email"
            required
          />
          <Campo
            etiqueta="Contraseña"
            type="password"
            value={datos.password}
            onChange={cambiar('password')}
            onBlur={() => setTocado(true)}
            autoComplete="new-password"
            error={errorPassword}
            ayuda={`Minimo ${MINIMO_PASSWORD} caracteres.`}
            required
          />
          <Boton type="submit" ocupado={ocupado}>
            {ocupado ? 'Creando la cuenta' : 'Crear cuenta'}
          </Boton>
        </form>

        <p className={estilos.pie}>
          ¿Ya tienes cuenta? <Link to="/login">Entra</Link>
        </p>
      </div>
    </div>
  )
}
