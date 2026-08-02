import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import * as api from '../api/auth.js'
import { registrarCierreDeSesion } from '../api/cliente.js'
import { borrarSesion, guardarSesion, haySesion } from '../api/sesion.js'

const AuthContexto = createContext(null)

// ProveedorAuth guarda quien esta dentro y expone las cuatro acciones de
// sesion. Es el unico sitio de la app que escribe o borra tokens.
export function ProveedorAuth({ children }) {
  const [usuario, setUsuario] = useState(null)

  // "cargando" arranca en true solo si hay un token guardado: sin token no hay
  // nada que comprobar y la pantalla de login puede pintarse de inmediato.
  const [cargando, setCargando] = useState(haySesion)
  const navegar = useNavigate()

  // Al abrir la app con un token guardado hay que confirmar que sigue sirviendo.
  // Se pide el perfil: si responde, la sesion es buena y de paso tenemos el
  // nombre y la moneda del usuario. Si no, el interceptor ya habra limpiado.
  useEffect(() => {
    if (!haySesion()) return

    let vigente = true
    api
      .perfil()
      .then((datos) => {
        if (vigente) setUsuario(datos)
      })
      .catch(() => {
        borrarSesion()
      })
      .finally(() => {
        if (vigente) setCargando(false)
      })

    return () => {
      vigente = false
    }
  }, [])

  const cerrarSesion = useCallback(() => {
    borrarSesion()
    setUsuario(null)
    navegar('/login', { replace: true })
  }, [navegar])

  // El cliente de axios no conoce react-router. Cuando el refresco falla avisa
  // por aqui y es este contexto el que decide a donde va el usuario.
  useEffect(() => {
    registrarCierreDeSesion(() => {
      setUsuario(null)
      navegar('/login', { replace: true })
    })
  }, [navegar])

  // entrar sirve para el login y para el registro: los dos responden lo mismo
  // (los dos tokens y el usuario) y los dos terminan con la sesion abierta.
  const entrar = useCallback((sesion) => {
    guardarSesion(sesion)
    setUsuario(sesion.usuario)
    return sesion.usuario
  }, [])

  const iniciarSesion = useCallback(
    (credenciales) => api.login(credenciales).then(entrar),
    [entrar],
  )

  const registrarse = useCallback((datos) => api.registro(datos).then(entrar), [entrar])

  const actualizarPerfil = useCallback(
    (datos) => api.actualizarPerfil(datos).then((actualizado) => {
      setUsuario(actualizado)
      return actualizado
    }),
    [],
  )

  const valor = useMemo(
    () => ({ usuario, cargando, iniciarSesion, registrarse, cerrarSesion, actualizarPerfil }),
    [usuario, cargando, iniciarSesion, registrarse, cerrarSesion, actualizarPerfil],
  )

  return <AuthContexto.Provider value={valor}>{children}</AuthContexto.Provider>
}

export function useAuth() {
  const contexto = useContext(AuthContexto)
  if (!contexto) throw new Error('useAuth se uso fuera de ProveedorAuth')
  return contexto
}
