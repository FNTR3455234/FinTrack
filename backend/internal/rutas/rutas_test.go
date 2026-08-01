package rutas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/config"
	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/middleware"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// bdFalsa hace de MongoDB para /health, sin base de datos de por medio.
type bdFalsa struct{}

func (bdFalsa) Ping(context.Context) error { return nil }

func routerDePrueba() *gin.Engine {
	cfg := &config.Config{
		GinModo:      gin.TestMode,
		CORSOrigenes: []string{"http://localhost:5173"},
	}
	return Configurar(cfg, Dependencias{BD: bdFalsa{}})
}

func pedir(metodo, ruta string) *httptest.ResponseRecorder {
	grabadora := httptest.NewRecorder()
	routerDePrueba().ServeHTTP(grabadora, httptest.NewRequest(metodo, ruta, nil))
	return grabadora
}

func TestConfigurar_RegistraHealthEnLaVersion1(t *testing.T) {
	grabadora := pedir(http.MethodGet, "/api/v1/health")

	assert.Equal(t, http.StatusOK, grabadora.Code)
}

func TestConfigurar_EngranaLosMiddlewares(t *testing.T) {
	grabadora := pedir(http.MethodGet, "/api/v1/health")

	// Si el id de peticion viaja de vuelta, la cadena de middlewares corrio.
	assert.NotEmpty(t, grabadora.Header().Get(middleware.CabeceraIDPeticion))
}

func TestConfigurar_UnaRutaInexistenteRespondeConElFormatoDeLaAPI(t *testing.T) {
	grabadora := pedir(http.MethodGet, "/api/v1/no-existe")

	assert.Equal(t, http.StatusNotFound, grabadora.Code)

	// Lo importante: NO es el "404 page not found" en texto plano de Gin.
	var cuerpo respuestas.SobreError
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, errores.CodigoRutaNoEncontrada, cuerpo.Error.Codigo)
	assert.Contains(t, cuerpo.Error.Mensaje, "/api/v1/no-existe")
}

func TestConfigurar_UnMetodoNoPermitidoResponde405(t *testing.T) {
	grabadora := pedir(http.MethodPost, "/api/v1/health")

	assert.Equal(t, http.StatusMethodNotAllowed, grabadora.Code)

	var cuerpo respuestas.SobreError
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, errores.CodigoMetodoNoPermitido, cuerpo.Error.Codigo)
	assert.Contains(t, cuerpo.Error.Mensaje, "POST")
}

func TestConfigurar_LaRaizNoRespondeNada(t *testing.T) {
	// Toda la API vive bajo /api/v1: la raiz no debe existir.
	grabadora := pedir(http.MethodGet, "/")

	assert.Equal(t, http.StatusNotFound, grabadora.Code)
}
