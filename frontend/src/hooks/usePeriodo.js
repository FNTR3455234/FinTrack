import { useCallback, useMemo, useState } from 'react'

// usePeriodo maneja el mes y el año que se estan mirando.
//
// El calculo de "mes siguiente" y "mes anterior" se hace con un Date sobre el
// dia 1 en vez de sumar y restar a mano con un if para diciembre y enero:
// menos codigo y sin el caso de borde de fin de año.
export function usePeriodo() {
  const ahora = new Date()
  const [periodo, setPeriodo] = useState({
    mes: ahora.getMonth() + 1,
    anio: ahora.getFullYear(),
  })

  const mover = useCallback((meses) => {
    setPeriodo((actual) => {
      const fecha = new Date(actual.anio, actual.mes - 1 + meses, 1)
      return { mes: fecha.getMonth() + 1, anio: fecha.getFullYear() }
    })
  }, [])

  const alMesActual = useCallback(() => {
    const hoy = new Date()
    setPeriodo({ mes: hoy.getMonth() + 1, anio: hoy.getFullYear() })
  }, [])

  // esMesActual apaga el boton de "Hoy" cuando ya estamos en el mes en curso.
  const esMesActual = useMemo(() => {
    const hoy = new Date()
    return periodo.mes === hoy.getMonth() + 1 && periodo.anio === hoy.getFullYear()
  }, [periodo])

  return { periodo, mover, alMesActual, esMesActual, setPeriodo }
}
