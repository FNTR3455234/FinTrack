package rutas

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// La especificacion la genera swaggo y vive en backend/docs. Estas pruebas no
// revisan su contenido campo por campo (eso lo hace `make swagger` al
// regenerarla), sino que este servida y que siga describiendo la API que de
// verdad existe.

func TestSwagger_SirveLaDocumentacion(t *testing.T) {
	router := routerDePrueba()

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))

	assert.Equal(t, http.StatusOK, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "swagger")
}

func TestSwagger_RedirigeALaPaginaPrincipal(t *testing.T) {
	router := routerDePrueba()

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/swagger", nil))

	require.Equal(t, http.StatusMovedPermanently, grabadora.Code)
	assert.Equal(t, "/swagger/index.html", grabadora.Header().Get("Location"))
}

func TestSwagger_LaEspecificacionDescribeLaAPI(t *testing.T) {
	router := routerDePrueba()

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	require.Equal(t, http.StatusOK, grabadora.Code)

	var especificacion struct {
		Info struct {
			Titulo string `json:"title"`
		} `json:"info"`
		BasePath            string                       `json:"basePath"`
		Paths               map[string]map[string]any    `json:"paths"`
		SecurityDefinitions map[string]map[string]string `json:"securityDefinitions"`
	}
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &especificacion))

	assert.Equal(t, "FinTrack API", especificacion.Info.Titulo)
	assert.Equal(t, "/api/v1", especificacion.BasePath)
	assert.Equal(t, "header", especificacion.SecurityDefinitions["BearerAuth"]["in"])

	// Si alguien agrega un endpoint y se olvida de anotarlo, esto no lo caza; lo
	// que si caza es que se borre backend/docs o que se regenere vacia.
	esperadas := []string{
		"/health", "/auth/login", "/auth/registro", "/auth/refresh", "/auth/perfil",
		"/cuentas", "/cuentas/{id}", "/categorias", "/categorias/{id}",
		"/transacciones", "/transacciones/{id}",
		"/transacciones/exportar", "/transacciones/importar",
		"/presupuestos", "/presupuestos/{id}",
		"/reportes/gastos-por-categoria", "/reportes/estado-presupuestos",
		"/reportes/resumen", "/reportes/tendencia", "/reportes/saldos",
	}
	for _, ruta := range esperadas {
		assert.Contains(t, especificacion.Paths, ruta, "falta documentar %s", ruta)
	}
}

func TestSwagger_NoSeCuelaEnElManejadorDeRutasDesconocidas(t *testing.T) {
	// /swagger cuelga de la raiz, fuera de /api/v1. Esta prueba fija que una
	// ruta parecida pero inventada siga cayendo en el 404 con formato JSON.
	router := routerDePrueba()

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/swaggerr", nil))

	assert.Equal(t, http.StatusNotFound, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "RUTA_NO_ENCONTRADA")
}
