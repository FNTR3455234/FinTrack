package servicios

import "time"

// MongoDB guarda las fechas con precision de milisegundos, mientras que
// time.Now() en Go trae nanosegundos.
//
// Si se guardara el valor de Go tal cual, la respuesta del POST traeria una
// hora con mas precision que la que quedo en la base, y un GET posterior
// devolveria un valor distinto para el mismo campo. Recortar aqui hace que lo
// que responde la API sea exactamente lo que esta guardado.

// ahora devuelve el instante actual en UTC, ya recortado a milisegundos.
func ahora() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

// diaCalendario normaliza la fecha de un movimiento al dia que el cliente quiso
// decir, anclado a las 12:00 UTC.
//
// La fecha de una transaccion es un DIA, no un instante: nadie apunta un gasto
// a las 19:03:22. Guardarla como instante trae el problema clasico de los husos
// horarios: un gasto del 31 de julio a las 19:00 en Ciudad de Mexico (UTC-6) es
// el 1 de agosto en UTC, y caeria en el presupuesto del mes equivocado.
//
// Date() devuelve el año, el mes y el dia en el huso con el que llego la fecha,
// asi que se toma el dia que vio el usuario y se vuelve a construir a mediodia
// UTC. El mediodia deja doce horas de margen a cada lado: el dia se lee igual en
// cualquier huso entre UTC-11 y UTC+11.
//
// Ver docs/decisiones.md, decision 019.
func diaCalendario(momento time.Time) time.Time {
	anio, mes, dia := momento.Date()
	return time.Date(anio, mes, dia, 12, 0, 0, 0, time.UTC)
}
