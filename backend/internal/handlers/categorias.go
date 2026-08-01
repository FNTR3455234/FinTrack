package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ServicioCategorias es lo que los handlers necesitan del servicio.
type ServicioCategorias interface {
	Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionCategoria) (*modelos.Categoria, error)
	Listar(ctx context.Context, usuarioID bson.ObjectID, tipo string, incluirArchivadas bool) ([]modelos.Categoria, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Categoria, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionCategoria) (*modelos.Categoria, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// Categorias agrupa los handlers de /categorias.
type Categorias struct {
	servicio ServicioCategorias
}

// NuevoCategorias construye el grupo de handlers.
func NuevoCategorias(servicio ServicioCategorias) *Categorias {
	return &Categorias{servicio: servicio}
}

// Listar atiende GET /categorias. Acepta ?tipo=ingreso|gasto y
// ?incluir_archivadas=true.
func (h *Categorias) Listar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	tipo := c.Query("tipo")
	if tipo != "" && tipo != modelos.TipoIngreso && tipo != modelos.TipoGasto {
		respuestas.Fallo(c, fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoDatosInvalidos,
			"El filtro tipo debe ser ingreso o gasto."))
		return
	}

	categorias, err := h.servicio.Listar(c.Request.Context(), usuarioID, tipo, booleano(c, "incluir_archivadas"))
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, categorias)
}

// Obtener atiende GET /categorias/:id.
func (h *Categorias) Obtener(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	categoria, err := h.servicio.PorID(c.Request.Context(), usuarioID, id)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, categoria)
}

// Crear atiende POST /categorias.
func (h *Categorias) Crear(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionCategoria
	if !enlazar(c, &peticion) {
		return
	}

	categoria, err := h.servicio.Crear(c.Request.Context(), usuarioID, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.Creado(c, categoria)
}

// Actualizar atiende PUT /categorias/:id.
func (h *Categorias) Actualizar(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}
	id, ok := idDeLaRuta(c)
	if !ok {
		return
	}

	var peticion modelos.PeticionCategoria
	if !enlazar(c, &peticion) {
		return
	}

	categoria, err := h.servicio.Actualizar(c.Request.Context(), usuarioID, id, peticion)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, categoria)
}

// Eliminar atiende DELETE /categorias/:id.
func (h *Categorias) Eliminar(c *gin.Context) {
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
