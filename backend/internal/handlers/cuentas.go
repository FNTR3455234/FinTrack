package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ServicioCuentas es lo que los handlers necesitan del servicio de cuentas.
type ServicioCuentas interface {
	Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionCuenta) (*modelos.Cuenta, error)
	Listar(ctx context.Context, usuarioID bson.ObjectID, incluirArchivadas bool) ([]modelos.Cuenta, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Cuenta, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionCuenta) (*modelos.Cuenta, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// Cuentas agrupa los handlers de /cuentas.
type Cuentas struct {
	servicio ServicioCuentas
}

// NuevoCuentas construye el grupo de handlers.
func NuevoCuentas(servicio ServicioCuentas) *Cuentas {
	return &Cuentas{servicio: servicio}
}

// Listar atiende GET /cuentas. Con ?incluir_archivadas=true devuelve tambien
// las archivadas.
//
//	@Summary		Listar cuentas
//	@Description	Solo las del usuario del token.
//	@Tags			cuentas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			incluir_archivadas	query	bool	false	"Incluye tambien las archivadas"
//	@Success		200	{object}	respuestas.Sobre{datos=[]modelos.Cuenta}
//	@Failure		401	{object}	respuestas.SobreError
//	@Router			/cuentas [get]
func (h *Cuentas) Listar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	cuentas, err := h.servicio.Listar(c.Request.Context(), usuarioID, booleano(c, "incluir_archivadas"))
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, cuentas)
}

// Obtener atiende GET /cuentas/:id.
//
//	@Summary		Obtener una cuenta
//	@Description	Responde 404 si el identificador es de otro usuario: un 403 confirmaria que existe.
//	@Tags			cuentas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Cuenta}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/cuentas/{id} [get]
func (h *Cuentas) Obtener(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	cuenta, err := h.servicio.PorID(c.Request.Context(), usuarioID, id)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, cuenta)
}

// Crear atiende POST /cuentas.
//
//	@Summary		Crear una cuenta
//	@Tags			cuentas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cuerpo	body	modelos.PeticionCuenta	true	"Datos de la cuenta"
//	@Success		201	{object}	respuestas.Sobre{datos=modelos.Cuenta}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Router			/cuentas [post]
func (h *Cuentas) Crear(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionCuenta
	if !enlazar(c, &peticion) {
		return
	}

	cuenta, err := h.servicio.Crear(c.Request.Context(), usuarioID, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Creado(c, cuenta)
}

// Actualizar atiende PUT /cuentas/:id.
//
//	@Summary		Editar una cuenta
//	@Tags			cuentas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Param			cuerpo	body	modelos.PeticionCuenta	true	"Datos de la cuenta"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Cuenta}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/cuentas/{id} [put]
func (h *Cuentas) Actualizar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionCuenta
	if !enlazar(c, &peticion) {
		return
	}

	cuenta, err := h.servicio.Actualizar(c.Request.Context(), usuarioID, id, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, cuenta)
}

// Eliminar atiende DELETE /cuentas/:id.
//
//	@Summary		Borrar una cuenta
//	@Description	Responde 409 CUENTA_CON_TRANSACCIONES si tiene movimientos. No hay borrado en cascada: archivala en su lugar.
//	@Tags			cuentas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Success		204	{string}	string	"Sin contenido"
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Failure		409	{object}	respuestas.SobreError
//	@Router			/cuentas/{id} [delete]
func (h *Cuentas) Eliminar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	if err := h.servicio.Eliminar(c.Request.Context(), usuarioID, id); err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.SinContenido(c)
}
