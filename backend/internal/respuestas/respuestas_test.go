package respuestas

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// contextoDePrueba arma un gin.Context aislado, sin levantar un servidor.
func contextoDePrueba() (*gin.Context, *httptest.ResponseRecorder) {
	grabadora := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(grabadora)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/prueba", nil)
	return c, grabadora
}

func TestOK_EnvuelveLosDatosSinMeta(t *testing.T) {
	c, grabadora := contextoDePrueba()

	OK(c, gin.H{"nombre": "Efectivo"})

	assert.Equal(t, http.StatusOK, grabadora.Code)

	var cuerpo map[string]any
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, map[string]any{"nombre": "Efectivo"}, cuerpo["datos"])
	// meta lleva omitempty: no debe aparecer en una respuesta sin paginacion.
	assert.NotContains(t, cuerpo, "meta")
}

func TestCreado_Responde201(t *testing.T) {
	c, grabadora := contextoDePrueba()

	Creado(c, gin.H{"id": "abc"})

	assert.Equal(t, http.StatusCreated, grabadora.Code)
}

func TestSinContenido_Responde204SinCuerpo(t *testing.T) {
	// Esta va por un router de verdad: c.Status solo marca el estado, y quien
	// lo escribe en la respuesta es Gin al terminar la cadena de handlers.
	router := gin.New()
	router.DELETE("/x", func(c *gin.Context) { SinContenido(c) })

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodDelete, "/x", nil))

	assert.Equal(t, http.StatusNoContent, grabadora.Code)
	assert.Empty(t, grabadora.Body.String())
}

func TestPaginado_CalculaElTotalDePaginasRedondeandoHaciaArriba(t *testing.T) {
	casos := []struct {
		nombre   string
		limite   int
		total    int64
		esperado int
	}{
		{"division exacta", 10, 100, 10},
		{"sobra una fila", 10, 101, 11},
		{"menos de una pagina", 10, 3, 1},
		{"sin resultados", 10, 0, 0},
		{"limite cero no divide entre cero", 0, 50, 0},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, grabadora := contextoDePrueba()

			Paginado(c, []string{}, 1, caso.limite, caso.total)

			var cuerpo Sobre
			require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
			require.NotNil(t, cuerpo.Meta)
			assert.Equal(t, caso.esperado, cuerpo.Meta.TotalPaginas)
			assert.Equal(t, caso.total, cuerpo.Meta.Total)
		})
	}
}

func TestFallo_UsaElCodigoYElEstadoDelErrorDeDominio(t *testing.T) {
	c, grabadora := contextoDePrueba()
	err := fintrackErrores.NoEncontrado(
		fintrackErrores.CodigoCategoriaNoEncontrada, "La categoria no existe.")

	Fallo(c, err.ConDetalles("id: 507f1f77bcf86cd799439011"))

	assert.Equal(t, http.StatusNotFound, grabadora.Code)

	var cuerpo SobreError
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, "CATEGORIA_NO_ENCONTRADA", cuerpo.Error.Codigo)
	assert.Equal(t, "La categoria no existe.", cuerpo.Error.Mensaje)
	assert.Equal(t, []string{"id: 507f1f77bcf86cd799439011"}, cuerpo.Error.Detalles)
}

func TestFallo_ConUnErrorCualquieraNoFiltraDetallesInternos(t *testing.T) {
	c, grabadora := contextoDePrueba()

	Fallo(c, errors.New("mongo: connection refused a 10.0.0.5:27017"))

	assert.Equal(t, http.StatusInternalServerError, grabadora.Code)
	// El detalle interno queda en la bitacora, nunca en la respuesta.
	assert.NotContains(t, grabadora.Body.String(), "10.0.0.5")

	var cuerpo SobreError
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, fintrackErrores.CodigoErrorInterno, cuerpo.Error.Codigo)
}

func TestFallo_AbortaLaCadenaDeMiddlewares(t *testing.T) {
	c, _ := contextoDePrueba()

	Fallo(c, fintrackErrores.NoAutorizado(fintrackErrores.CodigoTokenVencido, "Tu sesion expiro."))

	// Si no abortara, el handler siguiente escribiria una segunda respuesta.
	assert.True(t, c.IsAborted())
}
