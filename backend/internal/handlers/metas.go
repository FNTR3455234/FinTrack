package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ServicioMetas es lo que los handlers necesitan del servicio de metas.
type ServicioMetas interface {
	Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionMeta) (*modelos.Meta, error)
	Listar(ctx context.Context, usuarioID bson.ObjectID, f modelos.FiltroMetas) ([]modelos.ProgresoMeta, modelos.ResumenMetas, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.MetaConAportaciones, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionMeta) (*modelos.Meta, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
	Aportar(ctx context.Context, usuarioID, metaID bson.ObjectID, p modelos.PeticionAportacion) (*modelos.Aportacion, error)
	QuitarAportacion(ctx context.Context, usuarioID, metaID, aportacionID bson.ObjectID) error
}

// Metas agrupa los handlers de /metas.
type Metas struct {
	servicio ServicioMetas
}

// NuevoMetas construye el grupo de handlers.
func NuevoMetas(servicio ServicioMetas) *Metas {
	return &Metas{servicio: servicio}
}

// Listar atiende GET /metas.
//
//	@Summary		Listar metas de ahorro
//	@Description	Cada meta con lo que se lleva ahorrado, lo que falta, el porcentaje, el estado y a que ritmo mensual hay que ahorrar para llegar a tiempo. Cruza metas con aportaciones (consulta relacional 3).
//	@Tags			metas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			incluir_archivadas	query	bool	false	"Incluye tambien las archivadas"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.ListadoMetas}
//	@Failure		401	{object}	respuestas.SobreError
//	@Router			/metas [get]
func (h *Metas) Listar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	filtro := modelos.FiltroMetas{IncluirArchivadas: booleano(c, "incluir_archivadas")}

	metas, resumen, err := h.servicio.Listar(c.Request.Context(), usuarioID, filtro)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, modelos.ListadoMetas{Metas: metas, Resumen: resumen})
}

// Obtener atiende GET /metas/:id.
//
//	@Summary		Obtener una meta con sus aportaciones
//	@Description	El progreso mas el detalle de cada aportacion, que es lo que explica de donde sale la cifra. Responde 404 si la meta es de otro usuario.
//	@Tags			metas
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.MetaConAportaciones}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/metas/{id} [get]
func (h *Metas) Obtener(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	meta, err := h.servicio.PorID(c.Request.Context(), usuarioID, id)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, meta)
}
