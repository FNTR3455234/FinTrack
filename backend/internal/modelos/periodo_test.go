package modelos

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeriodoRango_VaDelPrimerDiaAlPrimeroDelMesSiguiente(t *testing.T) {
	inicio, fin := Periodo{Mes: 7, Anio: 2026}.Rango()

	assert.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), inicio)
	assert.Equal(t, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), fin)
}

func TestPeriodoRango_CierraElAñoEnDiciembre(t *testing.T) {
	inicio, fin := Periodo{Mes: 12, Anio: 2026}.Rango()

	assert.Equal(t, 2026, inicio.Year())
	assert.Equal(t, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), fin,
		"despues de diciembre viene enero del año siguiente")
}

func TestPeriodoRango_ElUltimoInstanteDelMesEntraEnElRango(t *testing.T) {
	// Febrero de 2028 es bisiesto: el rango tiene que llegar al dia 29.
	inicio, fin := Periodo{Mes: 2, Anio: 2028}.Rango()
	ultimo := time.Date(2028, 2, 29, 23, 59, 59, 0, time.UTC)

	assert.False(t, ultimo.Before(inicio))
	assert.True(t, ultimo.Before(fin), "el 29 de febrero tiene que caer dentro")
}

func TestPeriodoRetroceder(t *testing.T) {
	casos := map[string]struct {
		desde    Periodo
		meses    int
		esperado Periodo
	}{
		"dentro del mismo año":    {Periodo{Mes: 7, Anio: 2026}, 5, Periodo{Mes: 2, Anio: 2026}},
		"cruzando el año":         {Periodo{Mes: 2, Anio: 2026}, 5, Periodo{Mes: 9, Anio: 2025}},
		"un año entero":           {Periodo{Mes: 7, Anio: 2026}, 12, Periodo{Mes: 7, Anio: 2025}},
		"cero meses es el mismo":  {Periodo{Mes: 7, Anio: 2026}, 0, Periodo{Mes: 7, Anio: 2026}},
		"desde marzo, mes corto":  {Periodo{Mes: 3, Anio: 2026}, 1, Periodo{Mes: 2, Anio: 2026}},
		"desde enero hacia atras": {Periodo{Mes: 1, Anio: 2026}, 1, Periodo{Mes: 12, Anio: 2025}},
	}

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			assert.Equal(t, caso.esperado, caso.desde.Retroceder(caso.meses))
		})
	}
}

func TestPeriodoValido(t *testing.T) {
	casos := map[string]struct {
		periodo Periodo
		valido  bool
	}{
		"mes y año normales": {Periodo{Mes: 7, Anio: 2026}, true},
		"enero":              {Periodo{Mes: 1, Anio: 2026}, true},
		"diciembre":          {Periodo{Mes: 12, Anio: 2026}, true},
		"mes cero":           {Periodo{Mes: 0, Anio: 2026}, false},
		"mes trece":          {Periodo{Mes: 13, Anio: 2026}, false},
		"mes negativo":       {Periodo{Mes: -1, Anio: 2026}, false},
		"año muy viejo":      {Periodo{Mes: 7, Anio: 1999}, false},
		"año muy lejano":     {Periodo{Mes: 7, Anio: 2101}, false},
		"periodo vacio":      {Periodo{}, false},
	}

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			assert.Equal(t, caso.valido, caso.periodo.Valido())
		})
	}
}

func TestPeriodoDe_LeeLaFechaEnUTC(t *testing.T) {
	// Las 19:00 del 31 de julio en Ciudad de Mexico ya es 1 de agosto en UTC.
	// PeriodoDe trabaja en UTC a proposito, igual que Rango: los dos tienen que
	// estar de acuerdo o un movimiento caeria en dos meses distintos.
	enMexico := time.FixedZone("CST", -6*3600)
	momento := time.Date(2026, 7, 31, 19, 0, 0, 0, enMexico)

	assert.Equal(t, Periodo{Mes: 8, Anio: 2026}, PeriodoDe(momento))
}
