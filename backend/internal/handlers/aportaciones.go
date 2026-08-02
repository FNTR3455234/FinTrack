package handlers

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// Las aportaciones cuelgan de su meta: /metas/{id}/aportaciones.
//
// No son un recurso de primer nivel porque no existen por su cuenta. La ruta
// dice a que meta pertenecen, y por eso la peticion no lleva un meta_id en el
// cuerpo: si lo llevara, habria dos sitios donde decir lo mismo y podrian no
// coincidir.

// Aportar atiende POST /metas/:id/aportaciones.
//
//	@Summary		Registrar una aportacion
//	@Description	Dinero apartado para la meta. NO es una transaccion y no toca el saldo de ninguna cuenta: apartar dinero no es gastarlo. Se permite pasarse del objetivo.
//	@Tags			metas
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de la meta"
//	@Param			cuerpo	body	modelos.PeticionAportacion	true	"Monto, fecha y nota"
//	@Success		201	{object}	respuestas.Sobre{datos=modelos.Aportacion}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/metas/{id}/aportaciones [post]
func (h *Metas) Aportar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	metaID, ok := idDeLaRuta(c)
	if !ok {
		return
	}
	var peticion modelos.PeticionAportacion
	if !enlazar(c, &peticion) {
		return
	}

	aportacion, err := h.servicio.Aportar(c.Request.Context(), usuarioID, metaID, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Creado(c, aportacion)
}

// QuitarAportacion atiende DELETE /metas/:id/aportaciones/:aportacion.
//
//	@Summary		Borrar una aportacion
//	@Description	Responde 404 si la aportacion no es de esa meta, aunque exista en otra.
//	@Tags			metas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de la meta"
//	@Param			aportacion	path	string	true	"Identificador de la aportacion"
//	@Success		204	"Borrada"
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/metas/{id}/aportaciones/{aportacion} [delete]
func (h *Metas) QuitarAportacion(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	metaID, ok := idDeLaRuta(c)
	if !ok {
		return
	}
	aportacionID, ok := idDelTramo(c, "aportacion")
	if !ok {
		return
	}

	if err := h.servicio.QuitarAportacion(c.Request.Context(), usuarioID, metaID, aportacionID); err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.SinContenido(c)
}

// idDelTramo lee un parametro de ruta que no es ":id" y lo convierte a
// ObjectID. Lo necesita el sub-recurso, que tiene dos identificadores en la
// misma ruta.
func idDelTramo(c *gin.Context, nombre string) (bson.ObjectID, bool) {
	id, err := bson.ObjectIDFromHex(c.Param(nombre))
	if err != nil {
		respuestas.Fallo(c, fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoIDInvalido,
			"El identificador de la "+nombre+" no es valido."))
		return bson.NilObjectID, false
	}
	return id, true
}
