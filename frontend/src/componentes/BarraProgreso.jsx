import estilos from './BarraProgreso.module.css'

import { dinero, porcentaje } from '../utiles/formato.js'

// BarraProgreso muestra cuanto se lleva gastado de un presupuesto.
//
// El ancho se recorta al 100% aunque el porcentaje sea 140: una barra que se
// sale de su caja no dice "me pase mucho", dice "esto esta roto". Que se paso, y
// por cuanto, lo dicen el color y la cifra de al lado.
export default function BarraProgreso({ estado }) {
  const ancho = Math.min(estado.porcentaje_usado, 100)

  return (
    <div className={estilos.bloque}>
      <div className={estilos.encabezado}>
        <span className={estilos.nombre}>
          <span className={estilos.punto} style={{ background: estado.color }} aria-hidden="true" />
          {estado.nombre}
        </span>
        <span className={`${estilos.cifras} dinero`}>
          {dinero(estado.gastado)} <span className={estilos.tenue}>de {dinero(estado.monto_limite)}</span>
        </span>
      </div>

      {/*
        role="progressbar" con sus valores hace que un lector de pantalla lea
        "62 por ciento" en vez de saltarse un <div> vacio.
      */}
      <div
        className={estilos.riel}
        role="progressbar"
        aria-valuenow={Math.round(estado.porcentaje_usado)}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={`${estado.nombre}: ${porcentaje(estado.porcentaje_usado)} del presupuesto`}
      >
        <div className={estilos.relleno} data-estado={estado.estado} style={{ width: `${ancho}%` }} />
      </div>

      <div className={estilos.pie}>
        <span>{porcentaje(estado.porcentaje_usado)} usado</span>
        <span className="dinero" data-negativo={estado.disponible < 0 || undefined}>
          {estado.disponible < 0
            ? `${dinero(Math.abs(estado.disponible))} de mas`
            : `${dinero(estado.disponible)} disponibles`}
        </span>
      </div>
    </div>
  )
}
