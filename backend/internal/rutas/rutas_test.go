package rutas

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/config"
	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/middleware"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
	"github.com/FNTR3455234/FinTrack/backend/internal/servicios"
)

// Estas pruebas recorren la pila completa: router, middlewares, handlers,
// servicio y tokens de verdad. Lo unico falso es el repositorio, para no
// necesitar MongoDB.

// bdFalsa hace de MongoDB para /health.
type bdFalsa struct{}

func (bdFalsa) Ping(context.Context) error { return nil }

// repoUsuariosFalso guarda los usuarios en memoria.
type repoUsuariosFalso struct {
	porEmail map[string]*modelos.Usuario
	porID    map[bson.ObjectID]*modelos.Usuario
}

func nuevoRepoFalso() *repoUsuariosFalso {
	return &repoUsuariosFalso{
		porEmail: map[string]*modelos.Usuario{},
		porID:    map[bson.ObjectID]*modelos.Usuario{},
	}
}

func (r *repoUsuariosFalso) Crear(_ context.Context, u *modelos.Usuario) error {
	if _, existe := r.porEmail[u.Email]; existe {
		return repositorios.ErrDuplicado
	}
	u.ID = bson.NewObjectID()
	r.porEmail[u.Email] = u
	r.porID[u.ID] = u
	return nil
}

func (r *repoUsuariosFalso) PorEmail(_ context.Context, email string) (*modelos.Usuario, error) {
	if u, existe := r.porEmail[email]; existe {
		return u, nil
	}
	return nil, repositorios.ErrNoEncontrado
}

func (r *repoUsuariosFalso) PorID(_ context.Context, id bson.ObjectID) (*modelos.Usuario, error) {
	if u, existe := r.porID[id]; existe {
		return u, nil
	}
	return nil, repositorios.ErrNoEncontrado
}

func (r *repoUsuariosFalso) Actualizar(_ context.Context, id bson.ObjectID, nombre, moneda string) (*modelos.Usuario, error) {
	u, existe := r.porID[id]
	if !existe {
		return nil, repositorios.ErrNoEncontrado
	}
	u.Nombre, u.Moneda = nombre, moneda
	return u, nil
}

func routerDePrueba() *gin.Engine {
	cfg := &config.Config{GinModo: gin.TestMode, CORSOrigenes: []string{"http://localhost:5173"}}
	tokens := servicios.NuevoTokens(
		"secreto_de_acceso_para_pruebas_1234567890",
		"secreto_de_refresco_para_pruebas_09876543", 15, 7)

	return Configurar(cfg, Dependencias{
		BD:        bdFalsa{},
		Auth:      servicios.NuevoAuth(nuevoRepoFalso(), tokens),
		Validador: tokens,
	})
}

// --- Ayudantes --------------------------------------------------------------

func pedir(router *gin.Engine, metodo, ruta, token string, cuerpo any) *httptest.ResponseRecorder {
	var lector *bytes.Reader
	if cuerpo != nil {
		serializado, _ := json.Marshal(cuerpo)
		lector = bytes.NewReader(serializado)
	} else {
		lector = bytes.NewReader(nil)
	}

	peticion := httptest.NewRequest(metodo, ruta, lector)
	peticion.Header.Set("Content-Type", "application/json")
	if token != "" {
		peticion.Header.Set("Authorization", "Bearer "+token)
	}

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, peticion)
	return grabadora
}

// registrar da de alta un usuario y devuelve su sesion.
func registrar(t *testing.T, router *gin.Engine, email string) modelos.RespuestaSesion {
	t.Helper()

	grabadora := pedir(router, http.MethodPost, "/api/v1/auth/registro", "", modelos.PeticionRegistro{
		Nombre: "Usuario " + email, Email: email, Password: "Demo1234!",
	})
	require.Equal(t, http.StatusCreated, grabadora.Code, grabadora.Body.String())

	var cuerpo struct {
		Datos modelos.RespuestaSesion `json:"datos"`
	}
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	return cuerpo.Datos
}

func leerError(t *testing.T, grabadora *httptest.ResponseRecorder) respuestas.Detalle {
	t.Helper()
	var cuerpo respuestas.SobreError
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	return cuerpo.Error
}

// --- Rutas basicas ----------------------------------------------------------

func TestConfigurar_RegistraHealthEnLaVersion1(t *testing.T) {
	assert.Equal(t, http.StatusOK, pedir(routerDePrueba(), http.MethodGet, "/api/v1/health", "", nil).Code)
}

func TestConfigurar_EngranaLosMiddlewares(t *testing.T) {
	grabadora := pedir(routerDePrueba(), http.MethodGet, "/api/v1/health", "", nil)

	assert.NotEmpty(t, grabadora.Header().Get(middleware.CabeceraIDPeticion))
}

func TestConfigurar_UnaRutaInexistenteRespondeConElFormatoDeLaAPI(t *testing.T) {
	grabadora := pedir(routerDePrueba(), http.MethodGet, "/api/v1/no-existe", "", nil)

	assert.Equal(t, http.StatusNotFound, grabadora.Code)
	detalle := leerError(t, grabadora)
	assert.Equal(t, errores.CodigoRutaNoEncontrada, detalle.Codigo)
	assert.Contains(t, detalle.Mensaje, "/api/v1/no-existe")
}

func TestConfigurar_UnMetodoNoPermitidoResponde405(t *testing.T) {
	grabadora := pedir(routerDePrueba(), http.MethodPost, "/api/v1/health", "", nil)

	assert.Equal(t, http.StatusMethodNotAllowed, grabadora.Code)
	assert.Equal(t, errores.CodigoMetodoNoPermitido, leerError(t, grabadora).Codigo)
}

func TestConfigurar_LaRaizNoRespondeNada(t *testing.T) {
	assert.Equal(t, http.StatusNotFound, pedir(routerDePrueba(), http.MethodGet, "/", "", nil).Code)
}

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
