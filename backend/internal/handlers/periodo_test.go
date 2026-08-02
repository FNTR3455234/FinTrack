package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// contextoCon arma un contexto de Gin con la consulta indicada.
func contextoCon(consulta string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	grabadora := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(grabadora)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/reportes/resumen?"+consulta, nil)
	return c, grabadora
}

func TestPeriodoDeLaConsulta_LeeMesYAnio(t *testing.T) {
	c, _ := contextoCon("mes=7&anio=2026")

	periodo, ok := periodoDeLaConsulta(c)

	require.True(t, ok)
	assert.Equal(t, modelos.Periodo{Mes: 7, Anio: 2026}, periodo)
}

func TestPeriodoDeLaConsulta_SinParametrosUsaElMesEnCurso(t *testing.T) {
	c, _ := contextoCon("")
	ahora := time.Now().UTC()

	periodo, ok := periodoDeLaConsulta(c)

	require.True(t, ok)
	assert.Equal(t, modelos.Periodo{Mes: int(ahora.Month()), Anio: ahora.Year()}, periodo,
		"abrir el tablero sin escribir nada tiene que mostrar este mes")
}

func TestPeriodoDeLaConsulta_RechazaUnPeriodoImposible(t *testing.T) {
	// Un filtro de listado raro devuelve menos filas y se nota; un reporte del
	// mes 13 devolveria ceros que se leerian como "no gastaste nada".
	casos := []string{"mes=13&anio=2026", "mes=0&anio=2026", "mes=7&anio=1800", "mes=abc&anio=2026"}

	for _, consulta := range casos {
		t.Run(consulta, func(t *testing.T) {
			c, grabadora := contextoCon(consulta)

			_, ok := periodoDeLaConsulta(c)

			assert.False(t, ok)
			assert.Equal(t, http.StatusBadRequest, grabadora.Code)
			assert.Contains(t, grabadora.Body.String(), "PERIODO_INVALIDO")
		})
	}
}

func TestPeriodoDeLaConsulta_UnMesNoNumericoNoSeCuelaComoElMesActual(t *testing.T) {
	// entero() devuelve el valor por defecto cuando no puede convertir, asi que
	// "mes=abc" acabaria pareciendo el mes de hoy. Con anio fuera de rango se
	// comprueba que la validacion posterior si lo corta.
	c, grabadora := contextoCon("mes=abc&anio=99999")

	_, ok := periodoDeLaConsulta(c)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
}

func TestPeriodoOpcional_SinParametrosDevuelveNil(t *testing.T) {
	c, _ := contextoCon("")

	periodo, ok := periodoOpcional(c)

	require.True(t, ok)
	assert.Nil(t, periodo, "sin periodo, el listado de presupuestos los devuelve todos")
}

func TestPeriodoOpcional_ConMesYAnioDevuelveElPeriodo(t *testing.T) {
	c, _ := contextoCon("mes=7&anio=2026")

	periodo, ok := periodoOpcional(c)

	require.True(t, ok)
	require.NotNil(t, periodo)
	assert.Equal(t, modelos.Periodo{Mes: 7, Anio: 2026}, *periodo)
}

func TestPeriodoOpcional_ConSoloElMesCompletaElAñoActual(t *testing.T) {
	c, _ := contextoCon("mes=3")

	periodo, ok := periodoOpcional(c)

	require.True(t, ok)
	require.NotNil(t, periodo)
	assert.Equal(t, 3, periodo.Mes)
	assert.Equal(t, time.Now().UTC().Year(), periodo.Anio)
}

func TestPeriodoOpcional_TambienRechazaLoImposible(t *testing.T) {
	c, grabadora := contextoCon("mes=99")

	_, ok := periodoOpcional(c)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
}
