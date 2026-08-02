package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ServicioReportes es lo que los handlers necesitan del servicio de reportes.
type ServicioReportes interface {
	GastosPorCategoria(ctx context.Context, usuarioID bson.ObjectID, p modelos.Periodo) ([]modelos.GastoPorCategoria, error)
	EstadoPresupuestos(ctx context.Context, usuarioID bson.ObjectID, p modelos.Periodo) ([]modelos.EstadoPresupuesto, error)
	Resumen(ctx context.Context, usuarioID bson.ObjectID, p modelos.Periodo) (*modelos.Resumen, error)
	Tendencia(ctx context.Context, usuarioID bson.ObjectID, hasta modelos.Periodo, meses int) ([]modelos.PuntoTendencia, error)
	Saldos(ctx context.Context, usuarioID bson.ObjectID) ([]modelos.SaldoCuenta, error)
}

// Reportes agrupa los handlers de /reportes.
type Reportes struct {
	servicio ServicioReportes
}

// NuevoReportes construye el grupo de handlers.
func NuevoReportes(servicio ServicioReportes) *Reportes {
	return &Reportes{servicio: servicio}
}

// GastosPorCategoria atiende GET /reportes/gastos-por-categoria?mes=&anio=
// (consulta relacional 1).
func (h *Reportes) GastosPorCategoria(c *gin.Context) {
	usuarioID, periodo, ok := usuarioYPeriodo(c)
	if !ok {
		return
	}

	gastos, err := h.servicio.GastosPorCategoria(c.Request.Context(), usuarioID, periodo)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, gastos)
}

// EstadoPresupuestos atiende GET /reportes/estado-presupuestos?mes=&anio=
// (consulta relacional 2).
func (h *Reportes) EstadoPresupuestos(c *gin.Context) {
	usuarioID, periodo, ok := usuarioYPeriodo(c)
	if !ok {
		return
	}

	estados, err := h.servicio.EstadoPresupuestos(c.Request.Context(), usuarioID, periodo)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, estados)
}

// Resumen atiende GET /reportes/resumen?mes=&anio=, las cifras del tablero.
func (h *Reportes) Resumen(c *gin.Context) {
	usuarioID, periodo, ok := usuarioYPeriodo(c)
	if !ok {
		return
	}

	resumen, err := h.servicio.Resumen(c.Request.Context(), usuarioID, periodo)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, resumen)
}

// Tendencia atiende GET /reportes/tendencia?mes=&anio=&meses=, la serie que
// termina en el periodo pedido.
func (h *Reportes) Tendencia(c *gin.Context) {
	usuarioID, periodo, ok := usuarioYPeriodo(c)
	if !ok {
		return
	}

	meses := entero(c.Query("meses"), modelos.MesesTendenciaPorDefecto)

	serie, err := h.servicio.Tendencia(c.Request.Context(), usuarioID, periodo, meses)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, serie)
}

// Saldos atiende GET /reportes/saldos: cuanto queda hoy en cada cuenta.
func (h *Reportes) Saldos(c *gin.Context) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return
	}

	saldos, err := h.servicio.Saldos(c.Request.Context(), usuarioID)
	if err != nil {
		respuestas.Fallo(c, err)
		return
	}
	respuestas.OK(c, saldos)
}

// usuarioYPeriodo resuelve de una vez las dos cosas que necesitan casi todos
// los reportes, y ya responde el error si alguna falla.
func usuarioYPeriodo(c *gin.Context) (bson.ObjectID, modelos.Periodo, bool) {
	usuarioID, ok := usuarioAutenticado(c)
	if !ok {
		return bson.NilObjectID, modelos.Periodo{}, false
	}
	periodo, ok := periodoDeLaConsulta(c)
	if !ok {
		return bson.NilObjectID, modelos.Periodo{}, false
	}
	return usuarioID, periodo, true
}
