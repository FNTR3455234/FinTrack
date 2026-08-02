import { createContext, useCallback, useContext, useEffect, useState } from 'react'

const LLAVE = 'fintrack_tema'
const TemaContexto = createContext(null)

// temaInicial lee lo que ya dejo puesto el script en linea de index.html.
//
// Se lee del DOM y no de localStorage otra vez para que haya una sola fuente de
// verdad: si el script decidio "oscuro" por la preferencia del sistema, React
// arranca con lo mismo y no hay un fotograma con el tema equivocado.
function temaInicial() {
  return document.documentElement.dataset.tema === 'oscuro' ? 'oscuro' : 'claro'
}

export function ProveedorTema({ children }) {
  const [tema, setTema] = useState(temaInicial)

  // El atributo en <html> es lo que activa la paleta de tema.css. Escribirlo
  // aqui, en un efecto, mantiene el estado de React y el DOM sincronizados sin
  // que ningun componente tenga que acordarse de hacerlo.
  useEffect(() => {
    document.documentElement.dataset.tema = tema
    try {
      localStorage.setItem(LLAVE, tema)
    } catch {
      // Sin almacenamiento el tema dura lo que dure la pestaña.
    }
  }, [tema])

  const alternar = useCallback(() => {
    setTema((actual) => (actual === 'oscuro' ? 'claro' : 'oscuro'))
  }, [])

  return <TemaContexto.Provider value={{ tema, alternar }}>{children}</TemaContexto.Provider>
}

export function useTema() {
  const contexto = useContext(TemaContexto)
  if (!contexto) throw new Error('useTema se uso fuera de ProveedorTema')
  return contexto
}
