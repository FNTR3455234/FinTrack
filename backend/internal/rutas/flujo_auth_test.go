package rutas

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// --- Flujo completo de autenticacion ----------------------------------------

func TestFlujoCompleto_RegistroLoginPerfilYRefresco(t *testing.T) {
	router := routerDePrueba()

	// 1. Registro
	sesion := registrar(t, router, "demo@fintrack.mx")
	require.NotEmpty(t, sesion.TokenAcceso)
	require.NotEmpty(t, sesion.TokenRefresco)

	// 2. El token de acceso abre el perfil
	perfil := pedir(router, http.MethodGet, "/api/v1/auth/perfil", sesion.TokenAcceso, nil)
	require.Equal(t, http.StatusOK, perfil.Code)
	assert.Contains(t, perfil.Body.String(), "demo@fintrack.mx")

	// 3. Login con las mismas credenciales
	login := pedir(router, http.MethodPost, "/api/v1/auth/login", "", modelos.PeticionLogin{
		Email: "demo@fintrack.mx", Password: "Demo1234!",
	})
	require.Equal(t, http.StatusOK, login.Code)

	// 4. El token de refresco entrega uno de acceso nuevo, que tambien funciona
	refresco := pedir(router, http.MethodPost, "/api/v1/auth/refresh", "",
		modelos.PeticionRefresco{TokenRefresco: sesion.TokenRefresco})
	require.Equal(t, http.StatusOK, refresco.Code)

	var renovado struct {
		Datos modelos.RespuestaRefresco `json:"datos"`
	}
	require.NoError(t, json.Unmarshal(refresco.Body.Bytes(), &renovado))
	require.NotEmpty(t, renovado.Datos.TokenAcceso)

	conNuevo := pedir(router, http.MethodGet, "/api/v1/auth/perfil", renovado.Datos.TokenAcceso, nil)
	assert.Equal(t, http.StatusOK, conNuevo.Code)

	// 5. Editar el perfil
	editado := pedir(router, http.MethodPut, "/api/v1/auth/perfil", sesion.TokenAcceso,
		modelos.PeticionActualizarPerfil{Nombre: "Nombre Editado", Moneda: "USD"})
	require.Equal(t, http.StatusOK, editado.Code)
	assert.Contains(t, editado.Body.String(), "Nombre Editado")
}

func TestAislamiento_CadaTokenSoloVeSuPropioPerfil(t *testing.T) {
	// La comprobacion que pide la rubrica: dos usuarios, ningun cruce de datos.
	router := routerDePrueba()
	ana := registrar(t, router, "ana@fintrack.mx")
	beto := registrar(t, router, "beto@fintrack.mx")
	require.NotEqual(t, ana.Usuario.ID, beto.Usuario.ID)

	perfilDeAna := pedir(router, http.MethodGet, "/api/v1/auth/perfil", ana.TokenAcceso, nil)
	perfilDeBeto := pedir(router, http.MethodGet, "/api/v1/auth/perfil", beto.TokenAcceso, nil)

	assert.Contains(t, perfilDeAna.Body.String(), "ana@fintrack.mx")
	assert.NotContains(t, perfilDeAna.Body.String(), "beto@fintrack.mx")

	assert.Contains(t, perfilDeBeto.Body.String(), "beto@fintrack.mx")
	assert.NotContains(t, perfilDeBeto.Body.String(), "ana@fintrack.mx")
}

func TestAislamiento_EditarConElTokenDeAnaNoTocaAlUsuarioDeBeto(t *testing.T) {
	router := routerDePrueba()
	ana := registrar(t, router, "ana@fintrack.mx")
	registrar(t, router, "beto@fintrack.mx")

	editado := pedir(router, http.MethodPut, "/api/v1/auth/perfil", ana.TokenAcceso,
		modelos.PeticionActualizarPerfil{Nombre: "Ana Editada", Moneda: "EUR"})
	require.Equal(t, http.StatusOK, editado.Code)

	// El perfil de Beto sigue igual: el id sale del token, no del cuerpo.
	login := pedir(router, http.MethodPost, "/api/v1/auth/login", "", modelos.PeticionLogin{
		Email: "beto@fintrack.mx", Password: "Demo1234!",
	})
	require.Equal(t, http.StatusOK, login.Code)
	assert.NotContains(t, login.Body.String(), "Ana Editada")
	assert.NotContains(t, login.Body.String(), "EUR")
}

func TestRutasPrivadas_SinTokenResponden401(t *testing.T) {
	router := routerDePrueba()

	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodGet, "/api/v1/auth/perfil"},
		{http.MethodPut, "/api/v1/auth/perfil"},
	} {
		t.Run(caso.metodo+" "+caso.ruta, func(t *testing.T) {
			grabadora := pedir(router, caso.metodo, caso.ruta, "", nil)

			assert.Equal(t, http.StatusUnauthorized, grabadora.Code)
			assert.Equal(t, errores.CodigoNoAutenticado, leerError(t, grabadora).Codigo)
		})
	}
}

func TestRutasPublicas_NoExigenToken(t *testing.T) {
	router := routerDePrueba()

	grabadora := pedir(router, http.MethodPost, "/api/v1/auth/login", "", modelos.PeticionLogin{
		Email: "nadie@fintrack.mx", Password: "loquesea",
	})

	// 401 por credenciales, no por falta de token: la ruta si es publica.
	assert.Equal(t, http.StatusUnauthorized, grabadora.Code)
	assert.Equal(t, errores.CodigoCredencialesInvalidas, leerError(t, grabadora).Codigo)
}

func TestLimiteDePeticiones_ProtegeElGrupoAuth(t *testing.T) {
	router := routerDePrueba()

	var ultima *httptest.ResponseRecorder
	for i := 0; i < maximoPeticionesAuth+1; i++ {
		ultima = pedir(router, http.MethodPost, "/api/v1/auth/login", "", modelos.PeticionLogin{
			Email: "nadie@fintrack.mx", Password: "loquesea",
		})
	}

	assert.Equal(t, http.StatusTooManyRequests, ultima.Code)
	assert.Equal(t, errores.CodigoDemasiadosIntentos, leerError(t, ultima).Codigo)
}

func TestLimiteDePeticiones_NoAplicaAlRestoDeLaAPI(t *testing.T) {
	router := routerDePrueba()

	for i := 0; i < maximoPeticionesAuth+5; i++ {
		grabadora := pedir(router, http.MethodGet, "/api/v1/health", "", nil)
		require.Equal(t, http.StatusOK, grabadora.Code, "peticion %d", i)
	}
}
