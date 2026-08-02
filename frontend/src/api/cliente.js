import axios from 'axios'

import { deAxios } from './errores.js'
import { borrarSesion, guardarAcceso, tokenDeAcceso, tokenDeRefresco } from './sesion.js'

// Ruta base de la API. Relativa a proposito: en desarrollo la resuelve el proxy
// de Vite y en produccion la sirve el mismo origen que el bundle.
const BASE = import.meta.env.VITE_API_BASE || '/api/v1'

export const cliente = axios.create({
  baseURL: BASE,
  timeout: 15000,
  headers: { Accept: 'application/json' },
})

// El token de acceso vive 15 minutos. En vez de comprobar su vencimiento en el
// cliente (habria que decodificar el JWT y confiar en el reloj del navegador),
// se deja que la peticion falle con 401 y se renueva ahi. La fuente de verdad
// de si un token sirve es el servidor, no nosotros.
cliente.interceptors.request.use((config) => {
  const token = tokenDeAcceso()
  if (token && !config.sinToken) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// alCerrarSesion lo registra AuthContexto para poder sacar al usuario a la
// pantalla de login cuando el refresco ya no sirve. El cliente no importa
// react-router: solo avisa.
let alCerrarSesion = () => {}

export function registrarCierreDeSesion(callback) {
  alCerrarSesion = callback
}

// Refresco en curso. Si tres peticiones fallan con 401 a la vez, las tres se
// cuelgan de esta misma promesa en lugar de disparar tres refrescos: el segundo
// y el tercero llegarian con el token viejo y cerrarian la sesion sin motivo.
let refrescoEnCurso = null

function refrescar() {
  if (refrescoEnCurso) return refrescoEnCurso

  const refresco = tokenDeRefresco()
  if (!refresco) return Promise.reject(new Error('sin token de refresco'))

  refrescoEnCurso = cliente
    .post('/auth/refresh', { token_refresco: refresco }, { sinToken: true, sinReintento: true })
    .then((respuesta) => {
      const token = respuesta.data.datos.token_acceso
      guardarAcceso(token)
      return token
    })
    .finally(() => {
      refrescoEnCurso = null
    })

  return refrescoEnCurso
}

// Interceptor de respuesta: un solo reintento por peticion.
//
// La marca va en la propia config (reintentada), no en una variable del modulo:
// asi el limite es "una vez por peticion" y no "una vez en toda la sesion".
// Sin ella, un token de refresco vencido provocaria 401 -> refresh -> 401 ->
// refresh... en bucle.
cliente.interceptors.response.use(
  (respuesta) => respuesta,
  async (error) => {
    const peticion = error.config

    if (!debeReintentarse(error, peticion)) {
      return Promise.reject(deAxios(error))
    }

    peticion.reintentada = true
    try {
      const token = await refrescar()
      peticion.headers.Authorization = `Bearer ${token}`
      return await cliente(peticion)
    } catch {
      // El refresco tambien fallo: la sesion se acabo de verdad.
      borrarSesion()
      alCerrarSesion()
      return Promise.reject(deAxios(error))
    }
  },
)

// Rutas publicas de autenticacion. Su 401 significa "credenciales malas" o
// "token de refresco invalido", y renovar no arreglaria nada: solo convertiria
// un mensaje claro en un cierre de sesion.
//
// Se listan una por una y no con un prefijo /auth: /auth/perfil tambien empieza
// asi y ese si es un endpoint privado que debe reintentarse.
const RUTAS_SIN_REFRESCO = ['/auth/login', '/auth/registro', '/auth/refresh']

// debeReintentarse decide si este 401 merece un refresco.
function debeReintentarse(error, peticion) {
  if (error.response?.status !== 401) return false
  if (!peticion || peticion.reintentada || peticion.sinReintento) return false
  if (RUTAS_SIN_REFRESCO.includes(peticion.url)) return false
  return Boolean(tokenDeRefresco())
}
