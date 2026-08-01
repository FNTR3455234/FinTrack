package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// Las pruebas del contador van directo contra limitador y le pasan la hora,
// para no tener que esperar de verdad a que pase un minuto.

func TestLimitador_PermiteHastaElMaximoYCortaDespues(t *testing.T) {
	lim := &limitador{visitas: map[string]*ventana{}, maximo: 3, duracion: time.Minute}
	ahora := time.Now()

	for i := 1; i <= 3; i++ {
		permitido, _ := lim.registrar("10.0.0.1", ahora)
		assert.True(t, permitido, "la peticion %d debe pasar", i)
	}

	permitido, esperar := lim.registrar("10.0.0.1", ahora)

	assert.False(t, permitido, "la cuarta debe cortarse")
	assert.Positive(t, esperar, "debe decir cuanto falta para reintentar")
}

func TestLimitador_ReabreLaVentanaCuandoPasaElTiempo(t *testing.T) {
	lim := &limitador{visitas: map[string]*ventana{}, maximo: 2, duracion: time.Minute}
	ahora := time.Now()

	lim.registrar("10.0.0.1", ahora)
	lim.registrar("10.0.0.1", ahora)
	bloqueado, _ := lim.registrar("10.0.0.1", ahora)
	require.False(t, bloqueado)

	// Un minuto y un segundo despues, la ventana es nueva.
	permitido, _ := lim.registrar("10.0.0.1", ahora.Add(time.Minute+time.Second))

	assert.True(t, permitido)
}

func TestLimitador_CuentaCadaIPPorSeparado(t *testing.T) {
	lim := &limitador{visitas: map[string]*ventana{}, maximo: 1, duracion: time.Minute}
	ahora := time.Now()

	primeraDeUno, _ := lim.registrar("10.0.0.1", ahora)
	segundaDeUno, _ := lim.registrar("10.0.0.1", ahora)
	primeraDeOtro, _ := lim.registrar("10.0.0.2", ahora)

	assert.True(t, primeraDeUno)
	assert.False(t, segundaDeUno)
	assert.True(t, primeraDeOtro, "el limite de una IP no puede afectar a otra")
}

func TestLimitador_NoAcumulaVentanasVencidasEnMemoria(t *testing.T) {
	lim := &limitador{visitas: map[string]*ventana{}, maximo: 5, duracion: time.Minute}
	ahora := time.Now()

	// Cien IPs distintas en la primera ventana.
	for i := 0; i < 100; i++ {
		lim.registrar(string(rune('a'+i%26))+string(rune('0'+i/26)), ahora)
	}
	require.Len(t, lim.visitas, 100)

	// Al abrir una ventana nueva mucho despues, se limpian las viejas.
	lim.registrar("nueva-ip", ahora.Add(2*time.Minute))

	assert.Len(t, lim.visitas, 1, "solo debe quedar la ventana vigente")
}

func TestLimitePeticiones_Responde429ConRetryAfterYElFormatoDeLaAPI(t *testing.T) {
	router := gin.New()
	router.Use(LimitePeticiones(2, time.Minute))
	router.POST("/auth/login", func(c *gin.Context) { c.Status(http.StatusOK) })

	pedir := func() *httptest.ResponseRecorder {
		grabadora := httptest.NewRecorder()
		router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodPost, "/auth/login", nil))
		return grabadora
	}

	assert.Equal(t, http.StatusOK, pedir().Code)
	assert.Equal(t, http.StatusOK, pedir().Code)

	cortada := pedir()

	assert.Equal(t, http.StatusTooManyRequests, cortada.Code)
	assert.NotEmpty(t, cortada.Header().Get("Retry-After"))

	var cuerpo respuestas.SobreError
	require.NoError(t, json.Unmarshal(cortada.Body.Bytes(), &cuerpo))
	assert.Equal(t, errores.CodigoDemasiadosIntentos, cuerpo.Error.Codigo)
}

func TestLimitePeticiones_NoDejaLlegarAlHandlerCuandoCorta(t *testing.T) {
	llamadas := 0
	router := gin.New()
	router.Use(LimitePeticiones(1, time.Minute))
	router.POST("/auth/login", func(c *gin.Context) { llamadas++; c.Status(http.StatusOK) })

	for i := 0; i < 5; i++ {
		grabadora := httptest.NewRecorder()
		router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodPost, "/auth/login", nil))
	}

	assert.Equal(t, 1, llamadas, "solo la primera debe llegar al handler")
}
