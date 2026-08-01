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

// enUTC normaliza una fecha que llego del cliente con la misma regla.
func enUTC(momento time.Time) time.Time {
	return momento.UTC().Truncate(time.Millisecond)
}
