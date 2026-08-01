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
