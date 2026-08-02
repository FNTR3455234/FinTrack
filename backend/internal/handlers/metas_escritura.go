package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// Crear atiende POST /metas.
//
//	@Summary		Crear una meta de ahorro
//	@Description	La fecha limite es obligatoria: sin ella no se puede calcular a que ritmo hay que ahorrar.
//	@Tags			metas
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cuerpo	body	modelos.PeticionMeta	true	"Datos de la meta"
//	@Success		201	{object}	respuestas.Sobre{datos=modelos.Meta}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Router			/metas [post]
func (h *Metas) Crear(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	var peticion modelos.PeticionMeta
	if !enlazar(c, &peticion) {
		return
	}

	meta, err := h.servicio.Crear(c.Request.Context(), usuarioID, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Creado(c, meta)
}

// Actualizar atiende PUT /metas/:id.
//
//	@Summary		Actualizar una meta
//	@Description	Las aportaciones no se tocan: subir el objetivo no borra lo ya ahorrado, solo cambia el porcentaje.
//	@Tags			metas
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Param			cuerpo	body	modelos.PeticionMeta	true	"Datos de la meta"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Meta}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/metas/{id} [put]
func (h *Metas) Actualizar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}
	var peticion modelos.PeticionMeta
	if !enlazar(c, &peticion) {
		return
	}

	meta, err := h.servicio.Actualizar(c.Request.Context(), usuarioID, id, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, meta)
}

// Eliminar atiende DELETE /metas/:id.
//
//	@Summary		Borrar una meta
//	@Description	Borra tambien sus aportaciones: una aportacion no significa nada sin su meta. A diferencia de cuentas y categorias, aqui no hay 409.
//	@Tags			metas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Success		204	"Borrada"
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/metas/{id} [delete]
func (h *Metas) Eliminar(c *gin.Context) {
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
