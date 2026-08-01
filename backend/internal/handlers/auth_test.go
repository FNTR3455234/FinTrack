package handlers

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

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/middleware"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// servicioFalso implementa ServicioAuth y guarda lo que recibio, para poder
// comprobar que el handler le paso los datos correctos.
type servicioFalso struct {
	sesion    *modelos.RespuestaSesion
	refresco  *modelos.RespuestaRefresco
	usuario   *modelos.Usuario
	err       error
	recibioID bson.ObjectID
	recibio   any
}

func (s *servicioFalso) Registrar(_ context.Context, p modelos.PeticionRegistro) (*modelos.RespuestaSesion, error) {
	s.recibio = p
	return s.sesion, s.err
}

func (s *servicioFalso) IniciarSesion(_ context.Context, p modelos.PeticionLogin) (*modelos.RespuestaSesion, error) {
	s.recibio = p
	return s.sesion, s.err
}

func (s *servicioFalso) Refrescar(_ context.Context, p modelos.PeticionRefresco) (*modelos.RespuestaRefresco, error) {
	s.recibio = p
	return s.refresco, s.err
}

func (s *servicioFalso) Perfil(_ context.Context, id bson.ObjectID) (*modelos.Usuario, error) {
	s.recibioID = id
	return s.usuario, s.err
}

func (s *servicioFalso) ActualizarPerfil(_ context.Context, id bson.ObjectID, p modelos.PeticionActualizarPerfil) (*modelos.Usuario, error) {
	s.recibioID = id
	s.recibio = p
	return s.usuario, s.err
}

// routerAuth arma las rutas de /auth con el servicio falso. usuarioID simula lo
// que dejaria el middleware de autenticacion en las rutas privadas.
func routerAuth(servicio ServicioAuth, usuarioID *bson.ObjectID) *gin.Engine {
	ConfigurarValidador()
	auth := NuevoAuth(servicio)

	router := gin.New()
	router.POST("/auth/registro", auth.Registro)
	router.POST("/auth/login", auth.Login)
	router.POST("/auth/refresh", auth.Refrescar)

	privadas := router.Group("", func(c *gin.Context) {
		if usuarioID != nil {
			c.Set(middleware.ClaveUsuarioID, *usuarioID)
		}
		c.Next()
	})
	privadas.GET("/auth/perfil", auth.Perfil)
	privadas.PUT("/auth/perfil", auth.ActualizarPerfil)

	return router
}

func enviarJSON(router *gin.Engine, metodo, ruta string, cuerpo any) *httptest.ResponseRecorder {
	var lector *bytes.Reader
	switch v := cuerpo.(type) {
	case string:
		lector = bytes.NewReader([]byte(v))
	default:
		serializado, _ := json.Marshal(v)
		lector = bytes.NewReader(serializado)
	}

	peticion := httptest.NewRequest(metodo, ruta, lector)
	peticion.Header.Set("Content-Type", "application/json")

	grabadora := httptest.NewRecorder()
	router.ServeHTTP(grabadora, peticion)
	return grabadora
}

func leerError(t *testing.T, grabadora *httptest.ResponseRecorder) respuestas.Detalle {
	t.Helper()
	var cuerpo respuestas.SobreError
	require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
	return cuerpo.Error
}

// --- Registro ---------------------------------------------------------------

func TestRegistro_ConDatosValidosResponde201(t *testing.T) {
	servicio := &servicioFalso{sesion: &modelos.RespuestaSesion{TokenAcceso: "abc"}}
	router := routerAuth(servicio, nil)

	grabadora := enviarJSON(router, http.MethodPost, "/auth/registro", modelos.PeticionRegistro{
		Nombre: "Usuario Demo", Email: "demo@fintrack.mx", Password: "Demo1234!",
	})

	assert.Equal(t, http.StatusCreated, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "abc")
	assert.Equal(t, "demo@fintrack.mx", servicio.recibio.(modelos.PeticionRegistro).Email)
}

func TestRegistro_ReportaCadaCampoInvalidoConSuNombreEnEspañol(t *testing.T) {
	router := routerAuth(&servicioFalso{}, nil)

	grabadora := enviarJSON(router, http.MethodPost, "/auth/registro", map[string]any{
		"nombre":   "A",
		"email":    "no-es-un-correo",
		"password": "corta",
	})

	assert.Equal(t, http.StatusBadRequest, grabadora.Code)

	detalle := leerError(t, grabadora)
	assert.Equal(t, fintrackErrores.CodigoDatosInvalidos, detalle.Codigo)
	require.Len(t, detalle.Detalles, 3)
	// Los nombres son los del JSON, no los del struct de Go.
	assert.Contains(t, detalle.Detalles, "nombre: debe tener al menos 2 caracteres")
	assert.Contains(t, detalle.Detalles, "email: debe ser un correo electronico valido")
	assert.Contains(t, detalle.Detalles, "password: debe tener al menos 8 caracteres")
}

func TestRegistro_RechazaUnaContraseñaDeMasDe72Bytes(t *testing.T) {
	// bcrypt ignora todo lo que pase de 72 bytes: aceptarla daria una falsa
	// sensacion de seguridad.
	router := routerAuth(&servicioFalso{}, nil)
	larga := ""
	for i := 0; i < 80; i++ {
		larga += "a"
	}

	grabadora := enviarJSON(router, http.MethodPost, "/auth/registro", map[string]any{
		"nombre": "Usuario Demo", "email": "demo@fintrack.mx", "password": larga,
	})

	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
	assert.Contains(t, leerError(t, grabadora).Detalles[0], "password")
}

func TestRegistro_ConJSONMalFormadoResponde400ConOtroCodigo(t *testing.T) {
	router := routerAuth(&servicioFalso{}, nil)

	grabadora := enviarJSON(router, http.MethodPost, "/auth/registro", `{"nombre": "roto"`)

	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
	assert.Equal(t, fintrackErrores.CodigoJSONInvalido, leerError(t, grabadora).Codigo)
}

func TestRegistro_PropagaElErrorDelServicio(t *testing.T) {
	servicio := &servicioFalso{err: fintrackErrores.Conflicto(
		fintrackErrores.CodigoEmailYaRegistrado, "Ya existe una cuenta con ese correo.")}
	router := routerAuth(servicio, nil)

	grabadora := enviarJSON(router, http.MethodPost, "/auth/registro", modelos.PeticionRegistro{
		Nombre: "Usuario Demo", Email: "demo@fintrack.mx", Password: "Demo1234!",
	})

	assert.Equal(t, http.StatusConflict, grabadora.Code)
	assert.Equal(t, fintrackErrores.CodigoEmailYaRegistrado, leerError(t, grabadora).Codigo)
}

// --- Login y refresco -------------------------------------------------------

func TestLogin_ConCredencialesValidasResponde200(t *testing.T) {
	servicio := &servicioFalso{sesion: &modelos.RespuestaSesion{TokenAcceso: "token"}}
	router := routerAuth(servicio, nil)

	grabadora := enviarJSON(router, http.MethodPost, "/auth/login", modelos.PeticionLogin{
		Email: "demo@fintrack.mx", Password: "Demo1234!",
	})

	assert.Equal(t, http.StatusOK, grabadora.Code)
}

func TestLogin_ExigeCorreoYContraseña(t *testing.T) {
	router := routerAuth(&servicioFalso{}, nil)

	grabadora := enviarJSON(router, http.MethodPost, "/auth/login", map[string]any{})

	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
	assert.Len(t, leerError(t, grabadora).Detalles, 2)
}

func TestRefrescar_ExigeElTokenDeRefresco(t *testing.T) {
	router := routerAuth(&servicioFalso{}, nil)

	grabadora := enviarJSON(router, http.MethodPost, "/auth/refresh", map[string]any{})

	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
	assert.Contains(t, leerError(t, grabadora).Detalles, "token_refresco: es obligatorio")
}

func TestRefrescar_DevuelveElTokenNuevo(t *testing.T) {
	servicio := &servicioFalso{refresco: &modelos.RespuestaRefresco{TokenAcceso: "nuevo", ExpiraEn: 900}}
	router := routerAuth(servicio, nil)

	grabadora := enviarJSON(router, http.MethodPost, "/auth/refresh",
		modelos.PeticionRefresco{TokenRefresco: "el-de-refresco"})

	assert.Equal(t, http.StatusOK, grabadora.Code)
	assert.Contains(t, grabadora.Body.String(), "nuevo")
}

// --- Perfil -----------------------------------------------------------------

func TestPerfil_UsaElIDDelContextoYNoElDelCuerpo(t *testing.T) {
	// Es la regla central del aislamiento entre usuarios.
	delToken := bson.NewObjectID()
	servicio := &servicioFalso{usuario: &modelos.Usuario{ID: delToken, Email: "demo@fintrack.mx"}}
	router := routerAuth(servicio, &delToken)

	grabadora := enviarJSON(router, http.MethodGet, "/auth/perfil", nil)

	assert.Equal(t, http.StatusOK, grabadora.Code)
	assert.Equal(t, delToken, servicio.recibioID)
}

func TestPerfil_NoDevuelveElHashDeLaContraseña(t *testing.T) {
	usuarioID := bson.NewObjectID()
	servicio := &servicioFalso{usuario: &modelos.Usuario{
		ID: usuarioID, Email: "demo@fintrack.mx", Password: "$2a$10$hash-super-secreto",
	}}
	router := routerAuth(servicio, &usuarioID)

	grabadora := enviarJSON(router, http.MethodGet, "/auth/perfil", nil)

	assert.NotContains(t, grabadora.Body.String(), "hash-super-secreto")
	assert.NotContains(t, grabadora.Body.String(), "password")
}

func TestPerfil_SinUsuarioEnElContextoResponde401(t *testing.T) {
	router := routerAuth(&servicioFalso{}, nil)

	grabadora := enviarJSON(router, http.MethodGet, "/auth/perfil", nil)

	assert.Equal(t, http.StatusUnauthorized, grabadora.Code)
	assert.Equal(t, fintrackErrores.CodigoNoAutenticado, leerError(t, grabadora).Codigo)
}

func TestActualizarPerfil_ValidaYPasaLosDatosAlServicio(t *testing.T) {
	usuarioID := bson.NewObjectID()
	servicio := &servicioFalso{usuario: &modelos.Usuario{ID: usuarioID, Nombre: "Nuevo"}}
	router := routerAuth(servicio, &usuarioID)

	grabadora := enviarJSON(router, http.MethodPut, "/auth/perfil",
		modelos.PeticionActualizarPerfil{Nombre: "Nombre Nuevo", Moneda: "USD"})

	assert.Equal(t, http.StatusOK, grabadora.Code)
	assert.Equal(t, usuarioID, servicio.recibioID)
	assert.Equal(t, "USD", servicio.recibio.(modelos.PeticionActualizarPerfil).Moneda)
}

func TestActualizarPerfil_RechazaUnaMonedaQueNoSeanTresLetras(t *testing.T) {
	usuarioID := bson.NewObjectID()
	router := routerAuth(&servicioFalso{}, &usuarioID)

	grabadora := enviarJSON(router, http.MethodPut, "/auth/perfil", map[string]any{
		"nombre": "Nombre Nuevo", "moneda": "PESOS",
	})

	assert.Equal(t, http.StatusBadRequest, grabadora.Code)
	assert.Contains(t, leerError(t, grabadora).Detalles,
		"moneda: debe tener exactamente 3 caracteres")
}
