package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// Crear atiende POST /transacciones.
//
// La respuesta trae la transaccion y, cuando el gasto deja su categoria cerca
// del limite del mes, un campo "alerta" con el estado del presupuesto.
//
//	@Summary		Registrar un movimiento
//	@Description	Si el gasto deja su categoria al 80% o mas de su presupuesto del mes, la respuesta trae ademas el campo alerta.
//	@Tags			transacciones
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cuerpo	body	modelos.PeticionTransaccion	true	"Datos del movimiento"
//	@Success		201	{object}	respuestas.Sobre{datos=modelos.TransaccionCreada}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/transacciones [post]
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
//
//	@Summary		Editar un movimiento
//	@Description	El tipo tiene que seguir coincidiendo con el de su categoria.
//	@Tags			transacciones
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Param			cuerpo	body	modelos.PeticionTransaccion	true	"Datos del movimiento"
//	@Success		200	{object}	respuestas.Sobre{datos=modelos.Transaccion}
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/transacciones/{id} [put]
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
//
//	@Summary		Borrar un movimiento
//	@Description	Se borra de verdad: una transaccion no es referencia de nada mas.
//	@Tags			transacciones
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Identificador de 24 caracteres"
//	@Success		204	{string}	string	"Sin contenido"
//	@Failure		400	{object}	respuestas.SobreError
//	@Failure		401	{object}	respuestas.SobreError
//	@Failure		404	{object}	respuestas.SobreError
//	@Router			/transacciones/{id} [delete]
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
