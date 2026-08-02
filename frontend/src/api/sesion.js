// Guarda y lee los tokens de la sesion.
//
// Se usa localStorage y no una cookie porque la API es sin estado y espera el
// token en el encabezado Authorization: una cookie httpOnly seria mas segura
// contra XSS, pero exigiria que el backend la emitiera y la validara, y eso
// cambiaria el contrato de la fase 3.
//
// Queda anotado en docs/decisiones.md como limitacion conocida y aceptada para
// el alcance del proyecto.

const LLAVE_ACCESO = 'fintrack_token_acceso'
const LLAVE_REFRESCO = 'fintrack_token_refresco'

// Todo el acceso a localStorage pasa por aqui, envuelto en try: en modo
// incognito de Safari escribir lanza una excepcion, y una app que revienta al
// iniciar sesion por eso es peor que una que no recuerda la sesion.
function leer(llave) {
  try {
    return localStorage.getItem(llave)
  } catch {
    return null
  }
}

function escribir(llave, valor) {
  try {
    if (valor === null) {
      localStorage.removeItem(llave)
    } else {
      localStorage.setItem(llave, valor)
    }
  } catch {
    // Sin almacenamiento la sesion dura lo que dure la pestaña. Es degradado,
    // pero no es un fallo que valga la pena mostrarle al usuario.
  }
}

export function tokenDeAcceso() {
  return leer(LLAVE_ACCESO)
}

export function tokenDeRefresco() {
  return leer(LLAVE_REFRESCO)
}

// guardarSesion escribe lo que devuelven el registro y el login.
export function guardarSesion({ token_acceso, token_refresco }) {
  escribir(LLAVE_ACCESO, token_acceso)
  escribir(LLAVE_REFRESCO, token_refresco)
}

// guardarAcceso escribe solo el token de acceso, que es lo unico que renueva
// POST /auth/refresh: el de refresco sigue valido hasta que expire.
export function guardarAcceso(token) {
  escribir(LLAVE_ACCESO, token)
}

export function borrarSesion() {
  escribir(LLAVE_ACCESO, null)
  escribir(LLAVE_REFRESCO, null)
}

export function haySesion() {
  return Boolean(tokenDeAcceso())
}
