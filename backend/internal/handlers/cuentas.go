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
