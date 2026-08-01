package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
	"github.com/FNTR3455234/FinTrack/backend/internal/servicios"
)

const (
	secretoAcceso   = "secreto_de_acceso_para_pruebas_1234567890"
	secretoRefresco = "secreto_de_refresco_para_pruebas_09876543"
)

// routerProtegido arma una ruta que exige token y devuelve el usuario que vio.
func routerProtegido(validador ValidadorToken) (*gin.Engine, *bson.ObjectID) {
	var visto bson.ObjectID

	router := gin.New()
	router.GET("/privado", Autenticacion(validador), func(c *gin.Context) {
		id, ok := UsuarioID(c)
		require := map[bool]int{true: http.StatusOK, false: http.StatusInternalServerError}
		visto = id
		c.Status(require[ok])
	})
	return router, &visto
}

func pedirConToken(router *gin.Engine, encabezado string) *httptest.ResponseRecorder {
	peticion := httptest.NewRequest(http.MethodGet, "/privado", nil)
	if encabezado != "" {
		peticion.Header.Set("Authorization", encabezado)
	}
	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, peticion)
	return grabadora
}

func TestAutenticacion_ConUnTokenValidoDejaPasarYPublicaElUsuario(t *testing.T) {
	tokens := servicios.NuevoTokens(secretoAcceso, secretoRefresco, 15, 7)
	usuarioID := bson.NewObjectID()
	token, err := tokens.GenerarAcceso(usuarioID)
	require.NoError(t, err)

	router, visto := routerProtegido(tokens)
	grabadora := pedirConToken(router, "Bearer "+token)

	assert.Equal(t, http.StatusOK, grabadora.Code)
	// Este es el punto clave de todo el aislamiento entre usuarios: el id sale
	// del token, no del cuerpo ni de la query.
	assert.Equal(t, usuarioID, *visto)
}

func TestAutenticacion_AceptaElEsquemaBearerSinImportarMayusculas(t *testing.T) {
	tokens := servicios.NuevoTokens(secretoAcceso, secretoRefresco, 15, 7)
	token, err := tokens.GenerarAcceso(bson.NewObjectID())
	require.NoError(t, err)

	router, _ := routerProtegido(tokens)

	for _, esquema := range []string{"Bearer ", "bearer ", "BEARER "} {
		grabadora := pedirConToken(router, esquema+token)
		assert.Equal(t, http.StatusOK, grabadora.Code, "esquema %q", esquema)
	}
}

func TestAutenticacion_RechazaCuandoFaltaOEstaMalElEncabezado(t *testing.T) {
	tokens := servicios.NuevoTokens(secretoAcceso, secretoRefresco, 15, 7)
	router, _ := routerProtegido(tokens)

	casos := map[string]string{
		"sin encabezado":     "",
		"solo el token":      "un-token-suelto",
		"esquema equivocado": "Basic dXNlcjpwYXNz",
		"sin token":          "Bearer",
		"tres partes":        "Bearer token extra",
	}

	for nombre, encabezado := range casos {
		t.Run(nombre, func(t *testing.T) {
			grabadora := pedirConToken(router, encabezado)

			assert.Equal(t, http.StatusUnauthorized, grabadora.Code)

			var cuerpo respuestas.SobreError
			require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
			assert.Equal(t, errores.CodigoNoAutenticado, cuerpo.Error.Codigo)
		})
	}
}

func TestAutenticacion_DistingueUnTokenVencidoDeUnoInvalido(t *testing.T) {
	// El frontend usa esta diferencia: con TOKEN_VENCIDO intenta refrescar una
	// vez; con TOKEN_INVALIDO manda directo al login.
	vencidos := servicios.NuevoTokens(secretoAcceso, secretoRefresco, -1, 7)
	tokenVencido, err := vencidos.GenerarAcceso(bson.NewObjectID())
	require.NoError(t, err)

	tokens := servicios.NuevoTokens(secretoAcceso, secretoRefresco, 15, 7)
	router, _ := routerProtegido(tokens)

	t.Run("vencido", func(t *testing.T) {
		grabadora := pedirConToken(router, "Bearer "+tokenVencido)

		assert.Equal(t, http.StatusUnauthorized, grabadora.Code)
		var cuerpo respuestas.SobreError
		require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
		assert.Equal(t, errores.CodigoTokenVencido, cuerpo.Error.Codigo)
	})

	t.Run("invalido", func(t *testing.T) {
		grabadora := pedirConToken(router, "Bearer esto.no.sirve")

		assert.Equal(t, http.StatusUnauthorized, grabadora.Code)
		var cuerpo respuestas.SobreError
		require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
		assert.Equal(t, errores.CodigoTokenInvalido, cuerpo.Error.Codigo)
	})
}

func TestAutenticacion_NoDejaLlegarAlHandlerCuandoElTokenFalla(t *testing.T) {
	llego := false
	router := gin.New()
	router.GET("/privado",
		Autenticacion(servicios.NuevoTokens(secretoAcceso, secretoRefresco, 15, 7)),
		func(c *gin.Context) { llego = true })

	pedirConToken(router, "Bearer basura")

	assert.False(t, llego, "el handler no debe ejecutarse sin token valido")
}

func TestUsuarioID_DevuelveFalsoSiLaRutaNoPasoPorElMiddleware(t *testing.T) {
	grabadora := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(grabadora)

	_, ok := UsuarioID(c)

	assert.False(t, ok)
}
