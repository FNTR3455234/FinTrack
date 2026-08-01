package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ServicioTransacciones es lo que los handlers necesitan del servicio.
type ServicioTransacciones interface {
	Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionTransaccion) (*modelos.Transaccion, error)
	Listar(ctx context.Context, usuarioID bson.ObjectID, f modelos.FiltroTransacciones) ([]modelos.Transaccion, int64, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Transaccion, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionTransaccion) (*modelos.Transaccion, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// Transacciones agrupa los handlers de /transacciones.
type Transacciones struct {
	servicio ServicioTransacciones
}

// NuevoTransacciones construye el grupo de handlers.
func NuevoTransacciones(servicio ServicioTransacciones) *Transacciones {
	return &Transacciones{servicio: servicio}
}

// Listar atiende GET /transacciones con todos sus filtros:
// desde, hasta, tipo, categoria_id, cuenta_id, busqueda, pagina, limite y orden.
func (h *Transacciones) Listar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	filtro, ok := filtroDeTransacciones(c)
	if !ok {
		return
	}

	transacciones, total, err := h.servicio.Listar(c.Request.Context(), usuarioID, filtro)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Paginado(c, transacciones, filtro.Pagina, filtro.Limite, total)
}

// Obtener atiende GET /transacciones/:id.
func (h *Transacciones) Obtener(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	transaccion, err := h.servicio.PorID(c.Request.Context(), usuarioID, id)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, transaccion)
}

// Crear atiende POST /transacciones.
func (h *Transacciones) Crear(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionTransaccion
	if !enlazar(c, &peticion) {
		return
	}

	transaccion, err := h.servicio.Crear(c.Request.Context(), usuarioID, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Creado(c, transaccion)
}

// Actualizar atiende PUT /transacciones/:id.
func (h *Transacciones) Actualizar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionTransaccion
	if !enlazar(c, &peticion) {
		return
	}

	transaccion, err := h.servicio.Actualizar(c.Request.Context(), usuarioID, id, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, transaccion)
}

// Eliminar atiende DELETE /transacciones/:id.
func (h *Transacciones) Eliminar(c *gin.Context) {
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
