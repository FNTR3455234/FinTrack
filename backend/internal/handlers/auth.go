package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/middleware"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ServicioAuth es lo que los handlers de autenticacion necesitan del servicio.
type ServicioAuth interface {
	Registrar(ctx context.Context, p modelos.PeticionRegistro) (*modelos.RespuestaSesion, error)
	IniciarSesion(ctx context.Context, p modelos.PeticionLogin) (*modelos.RespuestaSesion, error)
	Refrescar(ctx context.Context, p modelos.PeticionRefresco) (*modelos.RespuestaRefresco, error)
	Perfil(ctx context.Context, usuarioID bson.ObjectID) (*modelos.Usuario, error)
	ActualizarPerfil(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionActualizarPerfil) (*modelos.Usuario, error)
}

// Auth agrupa los handlers de /auth.
type Auth struct {
	servicio ServicioAuth
}

// NuevoAuth construye el grupo de handlers.
func NuevoAuth(servicio ServicioAuth) *Auth {
	return &Auth{servicio: servicio}
}

// Registro atiende POST /auth/registro.
func (h *Auth) Registro(c *gin.Context) {
	var peticion modelos.PeticionRegistro
	if !enlazar(c, &peticion) {
		return
	}

	sesion, err := h.servicio.Registrar(c.Request.Context(), peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Creado(c, sesion)
}

// Login atiende POST /auth/login.
func (h *Auth) Login(c *gin.Context) {
	var peticion modelos.PeticionLogin
	if !enlazar(c, &peticion) {
		return
	}

	sesion, err := h.servicio.IniciarSesion(c.Request.Context(), peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, sesion)
}

// Refrescar atiende POST /auth/refresh.
func (h *Auth) Refrescar(c *gin.Context) {
	var peticion modelos.PeticionRefresco
	if !enlazar(c, &peticion) {
		return
	}

	renovado, err := h.servicio.Refrescar(c.Request.Context(), peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, renovado)
}

// Perfil atiende GET /auth/perfil.
func (h *Auth) Perfil(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	usuario, err := h.servicio.Perfil(c.Request.Context(), usuarioID)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, usuario)
}

// ActualizarPerfil atiende PUT /auth/perfil.
func (h *Auth) ActualizarPerfil(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionActualizarPerfil
	if !enlazar(c, &peticion) {
		return
	}

	usuario, err := h.servicio.ActualizarPerfil(c.Request.Context(), usuarioID, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, usuario)
}

// usuarioAutenticado saca el id que dejo el middleware de autenticacion.
//
// Si no esta, la ruta se registro sin ese middleware: es un error nuestro, no
// del cliente, asi que se responde 401 y queda en la bitacora como interno.
func usuarioAutenticado(c *gin.Context) (bson.ObjectID, bool) {
	usuarioID, ok := middleware.UsuarioID(c)
	if !ok {
		respuestas.Fallo(c, fintrackErrores.NoAutorizado(
			fintrackErrores.CodigoNoAutenticado, "No hay una sesion activa."))
		return bson.NilObjectID, false
	}
	return usuarioID, true
}
