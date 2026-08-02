import { useCallback, useEffect, useRef, useState } from 'react'

import { esCancelacion } from '../api/errores.js'

// usePeticion resuelve las cuatro cosas que toda pantalla que lee datos
// necesita: los datos, si esta cargando, el error y como volver a pedirlos.
//
// Sin este hook, cada pagina repetiria el mismo useEffect con sus tres useState
// y su try/catch, y bastaria olvidarse de uno para dejar un spinner eterno.
//
//   peticion    funcion que recibe { signal } y devuelve una promesa
//   claves      valores que, al cambiar, obligan a volver a pedir
export function usePeticion(peticion, claves = []) {
  const [datos, setDatos] = useState(null)
  const [cargando, setCargando] = useState(true)
  const [error, setError] = useState(null)
  const [intento, setIntento] = useState(0)

  // La funcion de peticion se guarda en una referencia porque cambia de
  // identidad en cada render. Si estuviera en las dependencias del efecto, el
  // efecto correria sin parar.
  const ultima = useRef(peticion)
  ultima.current = peticion

  useEffect(() => {
    // AbortController cancela la peticion si el componente se desmonta o si las
    // claves cambian antes de que llegue la respuesta. Sin esto, cambiar de mes
    // dos veces rapido puede dejar en pantalla el resultado del primer mes.
    const controlador = new AbortController()
    setCargando(true)
    setError(null)

    ultima
      .current({ signal: controlador.signal })
      .then((resultado) => {
        if (controlador.signal.aborted) return
        setDatos(resultado)
        setCargando(false)
      })
      .catch((fallo) => {
        if (controlador.signal.aborted || esCancelacion(fallo)) return
        setError(fallo)
        setCargando(false)
      })

    return () => controlador.abort()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...claves, intento])

  const recargar = useCallback(() => setIntento((n) => n + 1), [])

  return { datos, cargando, error, recargar }
}
