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
//
//	@Summary		Registrar un usuario
//	@Description	Da de alta la cuenta y devuelve la sesion ya iniciada.
//	@Tags			auth
//	@Produce		json
//	@Param			cuerpo	body	modelos.PeticionRegistro	true	"Datos del usuario"
//	@Success		201	{object}	respuestas.Sobre{datos=modelos.RespuestaSesion}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		409	{object}	respuestas.SobreError
//	@Failure		429	{object}	respuestas.SobreError
//	@Router			/auth/registro [post]
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
//
//	@Summary		Iniciar sesion
//	@Description	Devuelve el mismo error si el correo no existe o si la contraseña es incorrecta, para no revelar que correos estan registrados.
//	@Tags			auth
//	@Produce		json
//	@Param			cuerpo	body	modelos.PeticionLogin	true	"Correo y contraseña"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.RespuestaSesion}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		429	{object}	respuestas.SobreError
//	@Router			/auth/login [post]
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
//
//	@Summary		Renovar el token de acceso
//	@Description	Devuelve SOLO un token de acceso nuevo: el de refresco sigue valido hasta que expire, a los 7 dias.
//	@Tags			auth
//	@Produce		json
//	@Param			cuerpo	body	modelos.PeticionRefresco	true	"Token de refresco"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.RespuestaRefresco}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		429	{object}	respuestas.SobreError
//	@Router			/auth/refresh [post]
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
//
//	@Summary		Perfil del usuario
//	@Description	Datos del dueño del token.
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Usuario}
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/auth/perfil [get]
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
//
//	@Summary		Editar el perfil
//	@Description	Cambia el nombre y la moneda. El correo y la contraseña no se editan aqui.
//	@Tags			auth
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cuerpo	body	modelos.PeticionActualizarPerfil	true	"Nombre y moneda"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Usuario}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/auth/perfil [put]
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
