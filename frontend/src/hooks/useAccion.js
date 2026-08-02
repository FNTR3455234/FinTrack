import { useCallback, useState } from 'react'

// useAccion es a las escrituras lo que usePeticion a las lecturas: guarda si la
// accion esta en curso y el error que devolvio, para que el formulario sepa
// bloquear su boton y donde pintar el mensaje.
//
// Devuelve el resultado y no lanza: quien llama decide si cierra el modal
// mirando ese valor. Asi ningun formulario necesita su propio try/catch.
export function useAccion(accion) {
  const [ocupado, setOcupado] = useState(false)
  const [error, setError] = useState(null)

  const ejecutar = useCallback(
    async (...argumentos) => {
      setOcupado(true)
      setError(null)
      try {
        return { ok: true, datos: await accion(...argumentos) }
      } catch (fallo) {
        setError(fallo)
        return { ok: false, error: fallo }
      } finally {
        setOcupado(false)
      }
    },
    [accion],
  )

  const limpiar = useCallback(() => setError(null), [])

  return { ejecutar, ocupado, error, limpiar }
}
