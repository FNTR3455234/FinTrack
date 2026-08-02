package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ServicioPresupuestos es lo que los handlers necesitan del servicio.
type ServicioPresupuestos interface {
	Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionPresupuesto) (*modelos.Presupuesto, error)
	Listar(ctx context.Context, usuarioID bson.ObjectID, f modelos.FiltroPresupuestos) ([]modelos.Presupuesto, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Presupuesto, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionPresupuesto) (*modelos.Presupuesto, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// Presupuestos agrupa los handlers de /presupuestos.
type Presupuestos struct {
	servicio ServicioPresupuestos
}

// NuevoPresupuestos construye el grupo de handlers.
func NuevoPresupuestos(servicio ServicioPresupuestos) *Presupuestos {
	return &Presupuestos{servicio: servicio}
}

// Listar atiende GET /presupuestos. Con ?mes= y ?anio= devuelve solo los de ese
// periodo; sin ellos, todos.
//
//	@Summary		Listar presupuestos
//	@Description	Sin mes ni anio los devuelve todos; con ellos, solo los de ese periodo.
//	@Tags			presupuestos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			mes	query	int	false	"Mes del 1 al 12 (por defecto, el mes en curso)"
//	@Param			anio	query	int	false	"Año de 2000 a 2100 (por defecto, el año en curso)"
//	@Success		200	{object}	respuestas.Sobre{datos=[]modelos.Presupuesto}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Router			/presupuestos [get]
func (h *Presupuestos) Listar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	periodo, ok := periodoOpcional(c)
	if !ok {
		return
	}

	presupuestos, err := h.servicio.Listar(c.Request.Context(), usuarioID,
		modelos.FiltroPresupuestos{Periodo: periodo})
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, presupuestos)
}

// Obtener atiende GET /presupuestos/:id.
//
//	@Summary		Obtener un presupuesto
//	@Tags			presupuestos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Presupuesto}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/presupuestos/{id} [get]
func (h *Presupuestos) Obtener(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	presupuesto, err := h.servicio.PorID(c.Request.Context(), usuarioID, id)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, presupuesto)
}

// Crear atiende POST /presupuestos.
//
//	@Summary		Crear un presupuesto
//	@Description	Solo se presupuestan categorias de gasto. Un indice unico impide dos presupuestos para la misma categoria y mes (409).
//	@Tags			presupuestos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cuerpo	body	modelos.PeticionPresupuesto	true	"Categoria, limite y periodo"
//	@Success		201	{object}	respuestas.Sobre{datos=modelos.Presupuesto}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Failure		409	{object}	respuestas.SobreError
//	@Router			/presupuestos [post]
func (h *Presupuestos) Crear(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionPresupuesto
	if !enlazar(c, &peticion) {
		return
	}

	presupuesto, err := h.servicio.Crear(c.Request.Context(), usuarioID, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Creado(c, presupuesto)
}

// Actualizar atiende PUT /presupuestos/:id.
//
//	@Summary		Editar un presupuesto
//	@Description	Moverlo al periodo de otro que ya existe tambien es un duplicado (409).
//	@Tags			presupuestos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Param			cuerpo	body	modelos.PeticionPresupuesto	true	"Categoria, limite y periodo"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Presupuesto}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Failure		409	{object}	respuestas.SobreError
//	@Router			/presupuestos/{id} [put]
func (h *Presupuestos) Actualizar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionPresupuesto
	if !enlazar(c, &peticion) {
		return
	}

	presupuesto, err := h.servicio.Actualizar(c.Request.Context(), usuarioID, id, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, presupuesto)
}

// Eliminar atiende DELETE /presupuestos/:id.
//
//	@Summary		Borrar un presupuesto
//	@Tags			presupuestos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Success		204	{string}	string	"Sin contenido"
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/presupuestos/{id} [delete]
func (h *Presupuestos) Eliminar(c *gin.Context) {
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
