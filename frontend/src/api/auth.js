import { cliente } from './cliente.js'

// Llamadas a /auth.
//
// Todas devuelven ya el contenido de "datos": el sobre { datos, meta } es un
// detalle del transporte y no tiene por que llegar a los componentes.

export function registro(datos) {
  return cliente.post('/auth/registro', datos, { sinToken: true }).then((r) => r.data.datos)
}

export function login(datos) {
  return cliente.post('/auth/login', datos, { sinToken: true }).then((r) => r.data.datos)
}

export function perfil(opciones) {
  return cliente.get('/auth/perfil', opciones).then((r) => r.data.datos)
}

export function actualizarPerfil(datos) {
  return cliente.put('/auth/perfil', datos).then((r) => r.data.datos)
}
