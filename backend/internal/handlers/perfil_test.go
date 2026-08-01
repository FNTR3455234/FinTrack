package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

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
