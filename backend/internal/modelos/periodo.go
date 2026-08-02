package modelos

import "time"

// Periodo es un mes concreto de un año concreto: "julio de 2026".
//
// Los presupuestos y los reportes trabajan con periodos, no con fechas sueltas,
// porque la pregunta que responden siempre es del mes completo.
type Periodo struct {
	Mes  int `bson:"mes"  json:"mes"`
	Anio int `bson:"anio" json:"anio"`
}

// Rango devuelve el intervalo del periodo en UTC.
//
// El inicio entra y el fin NO: [1 de julio 00:00, 1 de agosto 00:00). Usar el
// primer instante del mes siguiente en vez del "ultimo del mes" evita tener que
// saber cuantos dias tiene febrero y evita el hueco clasico de perder los
// movimientos del dia 31 despues de las 00:00.
func (p Periodo) Rango() (time.Time, time.Time) {
	inicio := time.Date(p.Anio, time.Month(p.Mes), 1, 0, 0, 0, 0, time.UTC)
	return inicio, inicio.AddDate(0, 1, 0)
}

// Valido dice si el periodo existe en el calendario.
func (p Periodo) Valido() bool {
	return p.Mes >= 1 && p.Mes <= 12 && p.Anio >= AnioMinimo && p.Anio <= AnioMaximo
}

// Retroceder devuelve el periodo de hace `meses` meses.
//
// Se apoya en AddDate sobre el dia 1 para no pelearse con los meses de
// distinta longitud: restarle un mes al 31 de marzo daria el 3 de marzo.
func (p Periodo) Retroceder(meses int) Periodo {
	inicio, _ := p.Rango()
	anterior := inicio.AddDate(0, -meses, 0)
	return Periodo{Mes: int(anterior.Month()), Anio: anterior.Year()}
}

// PeriodoDe devuelve el periodo al que pertenece un instante, en UTC.
func PeriodoDe(momento time.Time) Periodo {
	utc := momento.UTC()
	return Periodo{Mes: int(utc.Month()), Anio: utc.Year()}
}

// Limites del año, iguales a los del $jsonSchema de MongoDB.
const (
	AnioMinimo = 2000
	AnioMaximo = 2100
)
