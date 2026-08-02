package servicios

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiaCalendario_GuardaElDiaQueVioElUsuario(t *testing.T) {
	// Este es el caso que justifica toda la funcion: un gasto de las 19:00 del
	// 31 de julio en Ciudad de Mexico (UTC-6) es 1 de agosto en UTC. Si se
	// guardara el instante tal cual, el movimiento contaria contra el
	// presupuesto de agosto y el usuario no entenderia por que.
	enMexico := time.FixedZone("CST", -6*3600)
	momento := time.Date(2026, 7, 31, 19, 0, 0, 0, enMexico)

	guardada := diaCalendario(momento)

	assert.Equal(t, time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC), guardada)
}

func TestDiaCalendario_AnclaSiempreAMediodiaUTC(t *testing.T) {
	casos := map[string]time.Time{
		"medianoche UTC":      time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		"mediodia UTC":        time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
		"casi medianoche UTC": time.Date(2026, 7, 3, 23, 59, 59, 0, time.UTC),
		"con nanosegundos":    time.Date(2026, 7, 3, 8, 15, 30, 123456789, time.UTC),
	}

	esperado := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	for nombre, momento := range casos {
		t.Run(nombre, func(t *testing.T) {
			assert.Equal(t, esperado, diaCalendario(momento),
				"cualquier hora del mismo dia se guarda igual")
		})
	}
}

func TestDiaCalendario_ElDiaSeLeeIgualEnCualquierHusoRazonable(t *testing.T) {
	// Con el ancla a mediodia quedan doce horas de margen a cada lado, asi que
	// la fecha se sigue leyendo como el dia 3 desde Tokio (UTC+9) o desde
	// Honolulu (UTC-10).
	guardada := diaCalendario(time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC))

	tokio := time.FixedZone("JST", 9*3600)
	honolulu := time.FixedZone("HST", -10*3600)

	assert.Equal(t, 3, guardada.In(tokio).Day())
	assert.Equal(t, 3, guardada.In(honolulu).Day())
}

func TestAhora_VieneRecortadoAMilisegundos(t *testing.T) {
	momento := ahora()

	assert.Equal(t, time.UTC, momento.Location())
	assert.Zero(t, momento.Nanosecond()%int(time.Millisecond),
		"MongoDB guarda milisegundos: sin recortar, la respuesta seria mas precisa que la base")
}
