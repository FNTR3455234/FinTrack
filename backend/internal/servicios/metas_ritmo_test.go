package servicios

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// dia arma una fecha de calendario, que es la unica precision con la que
// trabajan las metas.
func dia(anio int, mes time.Month, numero int) time.Time {
	return time.Date(anio, mes, numero, 12, 0, 0, 0, time.UTC)
}

func TestDiasHasta_NoDependeDeLaHoraDelDia(t *testing.T) {
	limite := dia(2026, time.September, 30)

	// El mismo dia a las 00:30 y a las 23:30 tienen que dar el mismo numero de
	// dias. Si se restaran los instantes tal cual, la division entera daria 60
	// en un caso y 59 en el otro.
	madrugada := time.Date(2026, time.August, 1, 0, 30, 0, 0, time.UTC)
	nocheCerrada := time.Date(2026, time.August, 1, 23, 30, 0, 0, time.UTC)

	assert.Equal(t, 60, diasHasta(madrugada, limite))
	assert.Equal(t, 60, diasHasta(nocheCerrada, limite))
}

func TestDiasHasta_EsNegativoCuandoLaFechaYaPaso(t *testing.T) {
	assert.Equal(t, -3, diasHasta(dia(2026, time.August, 4), dia(2026, time.August, 1)))
	assert.Equal(t, 0, diasHasta(dia(2026, time.August, 1), dia(2026, time.August, 1)))
}

func TestEstadoDeMeta_LoConseguidoManda(t *testing.T) {
	casos := []struct {
		nombre    string
		ahorrado  float64
		objetivo  float64
		dias      int
		esperado  string
		porQueAsi string
	}{
		{"todavia falta y queda tiempo", 5000, 20000, 45, modelos.EstadoMetaEnCurso, ""},
		{"llego justo al objetivo", 20000, 20000, 45, modelos.EstadoMetaCumplida, ""},
		{"se paso del objetivo", 22500, 20000, 45, modelos.EstadoMetaCumplida, ""},
		{"no llego y la fecha paso", 9000, 25000, -10, modelos.EstadoMetaVencida, ""},
		{
			"llego aunque la fecha ya paso", 25000, 25000, -10, modelos.EstadoMetaCumplida,
			"decir que fallo cuando junto el dinero seria mentirle",
		},
		{"el ultimo dia todavia cuenta", 5000, 20000, 0, modelos.EstadoMetaEnCurso, ""},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			estado := estadoDeMeta(modelos.ProgresoMeta{
				Ahorrado:      caso.ahorrado,
				MontoObjetivo: caso.objetivo,
				DiasRestantes: caso.dias,
			})
			assert.Equal(t, caso.esperado, estado, caso.porQueAsi)
		})
	}
}

func TestRitmoMensual_RepartePorLosMesesQueQuedan(t *testing.T) {
	// 11 500 en 90 dias son tres meses de 30: 3 833.33 al mes.
	assert.Equal(t, 3833.33, ritmoMensual(11500, 90))

	// 6 000 en 60 dias: 3 000 al mes.
	assert.Equal(t, 3000.0, ritmoMensual(6000, 60))
}

func TestRitmoMensual_LoQueYaNoDaTiempoSePideEntero(t *testing.T) {
	// Con menos de un mes por delante no se reparte: repartir 5 000 entre "medio
	// mes" daria 10 000 al mes, o peor, un numero pequeño que tranquiliza.
	assert.Equal(t, 5000.0, ritmoMensual(5000, 12))
	assert.Equal(t, 5000.0, ritmoMensual(5000, 0))

	// Una meta vencida sigue diciendo lo que falta, no un numero negativo.
	assert.Equal(t, 5000.0, ritmoMensual(5000, -20))
}

func TestRitmoMensual_EsCeroCuandoYaNoFaltaNada(t *testing.T) {
	assert.Zero(t, ritmoMensual(0, 60))
	assert.Zero(t, ritmoMensual(-500, 60))
}

func TestConCalendario_CompletaCadaFilaConLaFechaDeHoy(t *testing.T) {
	hoy := dia(2026, time.August, 2)
	progreso := []modelos.ProgresoMeta{
		{Nombre: "Viaje", MontoObjetivo: 25000, Ahorrado: 9000, Restante: 16000, FechaLimite: dia(2026, time.July, 28)},
		{Nombre: "Laptop", MontoObjetivo: 22000, Ahorrado: 22500, Restante: 0, FechaLimite: dia(2026, time.September, 28)},
	}

	completadas := conCalendario(progreso, hoy)

	assert.Equal(t, modelos.EstadoMetaVencida, completadas[0].Estado)
	assert.Equal(t, -5, completadas[0].DiasRestantes)
	assert.Equal(t, 16000.0, completadas[0].RitmoMensual, "vencida: lo que falta se pide entero")

	assert.Equal(t, modelos.EstadoMetaCumplida, completadas[1].Estado)
	assert.Equal(t, 57, completadas[1].DiasRestantes)
	assert.Zero(t, completadas[1].RitmoMensual)

	// No se toca la entrada: quien llama sigue teniendo su copia sin calendario.
	assert.Empty(t, progreso[0].Estado)
}

func TestResumirMetas_CuentaEstadosYSumaDinero(t *testing.T) {
	resumen := resumirMetas([]modelos.ProgresoMeta{
		{Estado: modelos.EstadoMetaCumplida, MontoObjetivo: 22000, Ahorrado: 22500},
		{Estado: modelos.EstadoMetaVencida, MontoObjetivo: 25000, Ahorrado: 9000},
		{Estado: modelos.EstadoMetaEnCurso, MontoObjetivo: 30000, Ahorrado: 18500},
	})

	assert.Equal(t, 3, resumen.Total)
	assert.Equal(t, 1, resumen.Cumplidas)
	assert.Equal(t, 1, resumen.Vencidas)
	assert.Equal(t, 77000.0, resumen.Objetivo)
	assert.Equal(t, 50000.0, resumen.Ahorrado)
}
