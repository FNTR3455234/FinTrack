import { useEffect, useState } from 'react'

// useRetardo devuelve el valor solo despues de que deje de cambiar durante un
// rato.
//
// Se usa en la caja de busqueda: sin esto, escribir "supermercado" lanzaria
// doce peticiones al servidor y las respuestas podrian llegar desordenadas.
// Con el retardo se lanza una sola, la del texto completo.
export function useRetardo(valor, milisegundos = 350) {
  const [retrasado, setRetrasado] = useState(valor)

  useEffect(() => {
    const temporizador = setTimeout(() => setRetrasado(valor), milisegundos)
    // Cada tecla cancela el temporizador anterior: por eso solo sobrevive el
    // ultimo, el de la pausa al terminar de escribir.
    return () => clearTimeout(temporizador)
  }, [valor, milisegundos])

  return retrasado
}
