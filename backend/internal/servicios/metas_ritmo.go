package servicios

import (
	"time"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Reloj devuelve la fecha de hoy. Es una funcion y no time.Now directamente
// para poder fijarlo en las pruebas: "faltan 45 dias" solo se puede comprobar si
// se sabe que dia es hoy.
type Reloj func() time.Time

// conCalendario completa los campos de una meta que dependen de la fecha de hoy.
//
// La agregacion de MongoDB suma el dinero; esto calcula el calendario. El
// reparto no es capricho: el dinero esta en la base y sumarlo alli evita traer
// todas las aportaciones, mientras que "cuantos dias faltan" depende de un
// instante que en las pruebas conviene poder elegir.
func conCalendario(progreso []modelos.ProgresoMeta, hoy time.Time) []modelos.ProgresoMeta {
	completadas := make([]modelos.ProgresoMeta, len(progreso))
	for i, meta := range progreso {
		meta.DiasRestantes = diasHasta(hoy, meta.FechaLimite)
		meta.Estado = estadoDeMeta(meta)
		meta.RitmoMensual = ritmoMensual(meta.Restante, meta.DiasRestantes)
		completadas[i] = meta
	}
	return completadas
}

// diasHasta cuenta los dias de calendario que quedan hasta la fecha limite.
//
// Las dos fechas se llevan a su dia a las 12:00 UTC antes de restar. Sin eso, el
// resultado dependeria de la hora: a las 23:00 de hoy "faltarian" 44,04 dias
// para algo que a las 08:00 estaba a 44,66, y la division entera daria un dia
// distinto segun cuando se preguntara.
//
// Puede ser negativo: una meta vencida hace tres dias devuelve -3, y eso es
// justo lo que hay que poder enseñar.
func diasHasta(hoy, limite time.Time) int {
	desde := diaCalendario(hoy)
	hasta := diaCalendario(limite)
	return int(hasta.Sub(desde).Hours() / 24)
}

// estadoDeMeta decide el semaforo.
//
// El orden importa: primero se mira si ya se junto el dinero. Una meta que
// llego al objetivo esta cumplida aunque la fecha ya haya pasado; lo contrario
// seria decirle a alguien que fallo cuando en realidad lo consiguio.
func estadoDeMeta(meta modelos.ProgresoMeta) string {
	switch {
	case meta.Ahorrado >= meta.MontoObjetivo:
		return modelos.EstadoMetaCumplida
	case meta.DiasRestantes < 0:
		return modelos.EstadoMetaVencida
	default:
		return modelos.EstadoMetaEnCurso
	}
}

// ritmoMensual dice cuanto habria que apartar cada mes para llegar a tiempo.
//
// Es la cifra que convierte una meta en un plan: "faltan 11 500" no dice si es
// mucho o poco, "unos 1 900 al mes" si.
//
// Devuelve 0 cuando ya no queda nada por juntar. Si la fecha ya paso, o queda
// menos de un mes, se devuelve el restante completo: lo que falta hay que
// juntarlo ya, y repartirlo entre "medio mes" daria una cifra tranquilizadora
// que no ayuda a nadie.
func ritmoMensual(restante float64, diasRestantes int) float64 {
	if restante <= 0 {
		return 0
	}
	if diasRestantes < modelos.DiasPorMes {
		return redondear(restante)
	}

	meses := float64(diasRestantes) / float64(modelos.DiasPorMes)
	return redondear(restante / meses)
}

// resumirMetas cuenta el conjunto para encabezar la pantalla.
func resumirMetas(progreso []modelos.ProgresoMeta) modelos.ResumenMetas {
	resumen := modelos.ResumenMetas{Total: len(progreso)}
	for _, meta := range progreso {
		switch meta.Estado {
		case modelos.EstadoMetaCumplida:
			resumen.Cumplidas++
		case modelos.EstadoMetaVencida:
			resumen.Vencidas++
		}
		resumen.Objetivo += meta.MontoObjetivo
		resumen.Ahorrado += meta.Ahorrado
	}

	// Se redondea al final y no en cada vuelta: sumar cifras ya redondeadas
	// arrastra el error de cada una.
	resumen.Objetivo = redondear(resumen.Objetivo)
	resumen.Ahorrado = redondear(resumen.Ahorrado)
	return resumen
}
