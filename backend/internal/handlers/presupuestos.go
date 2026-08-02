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
