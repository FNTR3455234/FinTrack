package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func TestIDPeticion_GeneraUnoCuandoElClienteNoLoManda(t *testing.T) {
	router := gin.New()
	router.Use(IDPeticion())

	var idVisto string
	router.GET("/x", func(c *gin.Context) {
		idVisto = c.GetString(ClaveIDPeticion)
		c.Status(http.StatusOK)
	})

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Len(t, idVisto, 16, "el id son 8 bytes en hexadecimal")
	// Tambien viaja de regreso, para que el cliente lo pueda reportar.
	assert.Equal(t, idVisto, grabadora.Header().Get(CabeceraIDPeticion))
}

func TestIDPeticion_RespetaElQueYaTraeLaPeticion(t *testing.T) {
	router := gin.New()
	router.Use(IDPeticion())
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	peticion := httptest.NewRequest(http.MethodGet, "/x", nil)
	peticion.Header.Set(CabeceraIDPeticion, "id-que-viene-del-proxy")

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, peticion)

	assert.Equal(t, "id-que-viene-del-proxy", grabadora.Header().Get(CabeceraIDPeticion))
}

func TestIDPeticion_DaUnIDDistintoACadaPeticion(t *testing.T) {
	router := gin.New()
	router.Use(IDPeticion())
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	vistos := make(map[string]bool)
	for i := 0; i < 50; i++ {
		grabadora := httptest.NewRecorder()
		router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/x", nil))
		vistos[grabadora.Header().Get(CabeceraIDPeticion)] = true
	}

	assert.Len(t, vistos, 50, "no debe repetirse ningun id")
}

func TestBitacora_NoInterrumpeLaCadenaYRegistraCadaPeticion(t *testing.T) {
	// La bitacora escribe con slog, que en las pruebas va al descarte. Lo que
	// importa comprobar es que deja pasar la peticion en los tres niveles
	// (info, warn, error) y no altera el cuerpo ni el estado.
	casos := []struct {
		nombre string
		estado int
	}{
		{"respuesta correcta", http.StatusOK},
		{"error del cliente", http.StatusBadRequest},
		{"error del servidor", http.StatusInternalServerError},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			router := gin.New()
			router.Use(IDPeticion(), Bitacora())
			router.GET("/x", func(c *gin.Context) { c.JSON(caso.estado, gin.H{"ok": true}) })

			grabadora := httptest.NewRecorder()
			router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/x?desde=2026-01-01", nil))

			assert.Equal(t, caso.estado, grabadora.Code)
			assert.JSONEq(t, `{"ok":true}`, grabadora.Body.String())
		})
	}
}

func TestRecuperacion_ConvierteUnPanicEnUn500ConElFormatoDeLaAPI(t *testing.T) {
	router := gin.New()
	router.Use(IDPeticion(), Recuperacion())
	router.GET("/explota", func(c *gin.Context) {
		panic("algo salio muy mal")
	})

	grabadora := httptest.NewRecorder()
	// Si el middleware no atrapara el panic, esta llamada tumbaria la prueba.
	require.NotPanics(t, func() {
		router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/explota", nil))
	})

	assert.Equal(t, http.StatusInternalServerError, grabadora.Code)

	var cuerpo respuestas.SobreError
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	assert.Equal(t, errores.CodigoErrorInterno, cuerpo.Error.Codigo)
	// Ni el texto del panic ni la traza pueden salir en la respuesta.
	assert.NotContains(t, grabadora.Body.String(), "algo salio muy mal")
}

func TestRecuperacion_NoEstorbaCuandoNoHayPanic(t *testing.T) {
	router := gin.New()
	router.Use(Recuperacion())
	router.GET("/bien", func(c *gin.Context) { respuestas.OK(c, gin.H{"ok": true}) })

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/bien", nil))

	assert.Equal(t, http.StatusOK, grabadora.Code)
}

func TestCORS_AutorizaUnOrigenDeLaLista(t *testing.T) {
	grabadora := pedirConOrigen(t, http.MethodGet, "http://localhost:5173")

	assert.Equal(t, "http://localhost:5173", grabadora.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, grabadora.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	assert.Equal(t, "true", grabadora.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", grabadora.Header().Get("Vary"))
}

func TestCORS_NoAutorizaUnOrigenQueNoEstaEnLaLista(t *testing.T) {
	grabadora := pedirConOrigen(t, http.MethodGet, "http://sitio-malicioso.com")

	// Sin el encabezado, el navegador bloquea la lectura de la respuesta.
	assert.Empty(t, grabadora.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, grabadora.Code, "la peticion se atiende igual; quien bloquea es el navegador")
}

func TestCORS_RespondeElPreflightSinLlegarAlHandler(t *testing.T) {
	llego := false
	router := gin.New()
	router.Use(CORS([]string{"http://localhost:5173"}))
	router.OPTIONS("/x", func(c *gin.Context) { llego = true })

	peticion := httptest.NewRequest(http.MethodOptions, "/x", nil)
	peticion.Header.Set("Origin", "http://localhost:5173")

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, peticion)

	assert.Equal(t, http.StatusNoContent, grabadora.Code)
	assert.False(t, llego, "el preflight se contesta en el middleware")
}

func TestCORS_ConComodinNoPermiteCredenciales(t *testing.T) {
	router := gin.New()
	router.Use(CORS([]string{"*"}))
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	peticion := httptest.NewRequest(http.MethodGet, "/x", nil)
	peticion.Header.Set("Origin", "http://cualquiera.com")

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, peticion)

	assert.Equal(t, "http://cualquiera.com", grabadora.Header().Get("Access-Control-Allow-Origin"))
	// El navegador rechaza la combinacion comodin + credenciales.
	assert.Empty(t, grabadora.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_SinEncabezadoOrigenNoAgregaNada(t *testing.T) {
	router := gin.New()
	router.Use(CORS([]string{"http://localhost:5173"}))
	router.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Empty(t, grabadora.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, grabadora.Code)
}

// pedirConOrigen hace una peticion al router de prueba desde el origen indicado.
func pedirConOrigen(t *testing.T, metodo, origen string) *httptest.ResponseRecorder {
	t.Helper()

	router := gin.New()
	router.Use(CORS([]string{"http://localhost:5173", "https://fintrack.mx"}))
	router.Handle(metodo, "/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	peticion := httptest.NewRequest(metodo, "/x", nil)
	peticion.Header.Set("Origin", origen)

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, peticion)
	return grabadora
}
