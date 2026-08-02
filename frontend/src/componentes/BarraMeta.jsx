import estilos from './BarraProgreso.module.css'

import { dinero, fechaCorta, porcentaje } from '../utiles/formato.js'

// BarraMeta muestra cuanto se lleva ahorrado de una meta.
//
// Comparte hoja de estilos con BarraProgreso —la barra se ve igual— pero es
// otro componente porque cuenta otra cosa: en un presupuesto llenar la barra es
// malo y en una meta es bueno, y el pie dice a que ritmo hay que ahorrar en vez
// de cuanto queda por gastar.
export default function BarraMeta({ meta, moneda }) {
  // Se recorta al 100 % aunque el porcentaje sea 116: una barra que se sale de
  // su caja no dice "junte de mas", dice "esto esta roto".
  const ancho = Math.min(meta.porcentaje, 100)

  return (
    <div className={estilos.bloque}>
      <div className={estilos.encabezado}>
        <span className={estilos.nombre}>
          <span className={estilos.punto} style={{ background: meta.color }} aria-hidden="true" />
          {meta.nombre}
        </span>
        <span className={`${estilos.cifras} dinero`}>
          {dinero(meta.ahorrado, moneda)}{' '}
          <span className={estilos.tenue}>de {dinero(meta.monto_objetivo, moneda)}</span>
        </span>
      </div>

      <div
        className={estilos.riel}
        role="progressbar"
        aria-valuenow={Math.round(meta.porcentaje)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={`${meta.nombre}: ${porcentaje(meta.porcentaje)} del objetivo`}
      >
        <div className={estilos.relleno} data-estado={meta.estado} style={{ width: `${ancho}%` }} />
      </div>

      <div className={estilos.pie}>
        <span>{porcentaje(meta.porcentaje)} juntado</span>
        <span className="dinero" data-negativo={meta.estado === 'vencida' || undefined}>
          {textoDelPie(meta, moneda)}
        </span>
      </div>
    </div>
  )
}

// textoDelPie dice lo unico que hace falta saber de cada meta, segun como vaya.
//
// El ritmo es lo que convierte una meta en un plan: "faltan 14 500" no dice si
// es mucho o poco; "unos 4 833 al mes" si.
function textoDelPie(meta, moneda) {
  if (meta.estado === 'cumplida') {
    return `Lista desde antes del ${fechaCorta(meta.fecha_limite)}`
  }
  if (meta.estado === 'vencida') {
    return `Vencio el ${fechaCorta(meta.fecha_limite)}, faltaron ${dinero(meta.restante, moneda)}`
  }
  return `${dinero(meta.ritmo_mensual, moneda)} al mes · ${diasEnPalabras(meta.dias_restantes)}`
}

// diasEnPalabras evita el "1 dias" y el "0 dias" que quedan raros al leerlos.
function diasEnPalabras(dias) {
  if (dias === 0) return 'vence hoy'
  if (dias === 1) return 'queda 1 dia'
  return `quedan ${dias} dias`
}
