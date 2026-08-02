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
	Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionTransaccion) (*modelos.TransaccionCreada, error)
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
//
//	@Summary		Listar transacciones
//	@Description	Listado paginado con filtros. La meta trae pagina, limite, total y total_paginas.
//	@Tags			transacciones
//	@Produce		json
//	@Security		BearerAuth
//	@Param			desde	query	string	false	"Fecha inicial AAAA-MM-DD"
//	@Param			hasta	query	string	false	"Fecha final AAAA-MM-DD, incluida"
//	@Param			tipo	query	string	false	"ingreso o gasto"	Enums(ingreso, gasto)
//	@Param			categoria_id	query	string	false	"Filtra por categoria"
//	@Param			cuenta_id	query	string	false	"Filtra por cuenta"
//	@Param			busqueda	query	string	false	"Texto en descripcion o notas"
//	@Param			orden	query	string	false	"Orden del listado"	Enums(fecha_desc, fecha_asc, monto_desc, monto_asc)
//	@Param			pagina	query	int	false	"Pagina, desde 1"
//	@Param			limite	query	int	false	"Resultados por pagina, de 1 a 100"
//	@Success		200	{object}	respuestas.Sobre{datos=[]modelos.Transaccion,meta=respuestas.Meta}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Router			/transacciones [get]
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
//
//	@Summary		Obtener una transaccion
//	@Tags			transacciones
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Transaccion}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/transacciones/{id} [get]
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
