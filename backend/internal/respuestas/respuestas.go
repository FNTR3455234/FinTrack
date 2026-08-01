// Package respuestas escribe TODAS las respuestas JSON de la API con el mismo
// formato. Esta en su propio paquete porque lo usan tanto los handlers como los
// middlewares (el de autenticacion tambien tiene que responder un 401 con la
// misma forma).
//
//	Exito:  { "datos": ..., "meta": {...} }
//	Error:  { "error": { "codigo": "...", "mensaje": "...", "detalles": [...] } }
package respuestas

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
)

// Sobre es el cuerpo de una respuesta exitosa. Meta se omite cuando no aplica.
type Sobre struct {
	Datos any   `json:"datos"`
	Meta  *Meta `json:"meta,omitempty"`
}

// Meta acompaña a los listados paginados.
type Meta struct {
	Pagina       int   `json:"pagina"`
	Limite       int   `json:"limite"`
	Total        int64 `json:"total"`
	TotalPaginas int   `json:"total_paginas"`
}

// SobreError es el cuerpo de una respuesta con error.
type SobreError struct {
	Error Detalle `json:"error"`
}

// Detalle describe el error para el cliente. Nunca lleva informacion interna.
type Detalle struct {
	Codigo   string   `json:"codigo"`
	Mensaje  string   `json:"mensaje"`
	Detalles []string `json:"detalles,omitempty"`
}

// OK responde 200 con el cuerpo indicado.
func OK(c *gin.Context, datos any) {
	c.JSON(http.StatusOK, Sobre{Datos: datos})
}

// Creado responde 201, para los POST que crean un recurso.
func Creado(c *gin.Context, datos any) {
	c.JSON(http.StatusCreated, Sobre{Datos: datos})
}

// SinContenido responde 204, para los DELETE.
func SinContenido(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Paginado responde 200 con los datos y su meta de paginacion.
func Paginado(c *gin.Context, datos any, pagina, limite int, total int64) {
	totalPaginas := 0
	if limite > 0 {
		// Division entera redondeando hacia arriba.
		totalPaginas = int((total + int64(limite) - 1) / int64(limite))
	}
	c.JSON(http.StatusOK, Sobre{
		Datos: datos,
		Meta:  &Meta{Pagina: pagina, Limite: limite, Total: total, TotalPaginas: totalPaginas},
	})
}

// Fallo traduce cualquier error a la respuesta JSON que le corresponde.
//
// Si es un error de dominio usa su codigo y su HTTP; si no, se registra
// completo en la bitacora y al cliente se le devuelve un 500 generico, para no
// filtrar detalles internos.
func Fallo(c *gin.Context, err error) {
	dominio, esDominio := errores.Como(err)
	if !esDominio {
		dominio = errores.Interno(err)
	}

	// Los 5xx siempre se registran; los 4xx son culpa del cliente y solo se
	// anotan en nivel de depuracion para no llenar la bitacora.
	registro := slog.With("codigo", dominio.Codigo, "ruta", c.Request.URL.Path)
	if dominio.HTTP >= http.StatusInternalServerError {
		registro.Error("error interno", "detalle", dominio.Error())
	} else {
		registro.Debug("error de cliente", "detalle", dominio.Error())
	}

	c.AbortWithStatusJSON(dominio.HTTP, SobreError{Detalle{
		Codigo:   dominio.Codigo,
		Mensaje:  dominio.Mensaje,
		Detalles: dominio.Detalles,
	}})
}
