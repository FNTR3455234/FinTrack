package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// bdFalsa implementa VerificadorBD sin MongoDB: devuelve el error que se le
// indique. Es todo lo que hace falta para probar los dos caminos del handler.
type bdFalsa struct{ err error }

func (b bdFalsa) Ping(context.Context) error { return b.err }

func ejecutarSalud(bd VerificadorBD) *httptest.ResponseRecorder {
	router := gin.New()
	router.GET("/api/v1/health", Salud(bd, "1.2.3"))

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	return grabadora
}

func TestSalud_ConLaBaseArribaResponde200(t *testing.T) {
	grabadora := ejecutarSalud(bdFalsa{err: nil})

	assert.Equal(t, http.StatusOK, grabadora.Code)

	var cuerpo struct {
		Datos EstadoSalud `json:"datos"`
	}
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, "ok", cuerpo.Datos.Estado)
	assert.Equal(t, "ok", cuerpo.Datos.Mongo)
	assert.Equal(t, "1.2.3", cuerpo.Datos.Version)
	assert.False(t, cuerpo.Datos.Hora.IsZero())
}

func TestSalud_ConLaBaseCaidaResponde503(t *testing.T) {
	grabadora := ejecutarSalud(bdFalsa{err: errors.New("connection refused")})

	// 503 y no 500: le dice al orquestador que deje de mandarle trafico.
	assert.Equal(t, http.StatusServiceUnavailable, grabadora.Code)

	var cuerpo struct {
		Datos EstadoSalud `json:"datos"`
	}
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, "degradado", cuerpo.Datos.Estado)
	assert.Equal(t, "sin_respuesta", cuerpo.Datos.Mongo)
	// El detalle del error de Mongo no se filtra al cliente.
	assert.NotContains(t, grabadora.Body.String(), "connection refused")
}

func TestSalud_RespetaElFormatoUniformeDeRespuesta(t *testing.T) {
	grabadora := ejecutarSalud(bdFalsa{})

	var sobre respuestas.Sobre
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &sobre))
	assert.NotNil(t, sobre.Datos)
	assert.Nil(t, sobre.Meta)
}
