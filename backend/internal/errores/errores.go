// Package errores define los errores de dominio de FinTrack.
//
// La idea es que los servicios devuelvan errores que ya saben que codigo HTTP
// y que codigo de negocio les corresponde, y que los handlers solo tengan que
// pasarlos a la capa de respuesta. Asi ningun handler decide "esto es un 404":
// esa decision vive junto a la regla que la produjo.
package errores

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorDominio es un error de negocio con toda la informacion que necesita la
// respuesta HTTP. Codigo es el identificador estable que consume el frontend
// (por ejemplo CATEGORIA_NO_ENCONTRADA); Mensaje es el texto para la persona.
type ErrorDominio struct {
	Codigo   string
	Mensaje  string
	HTTP     int
	Detalles []string
	// Causa guarda el error original (de Mongo, por ejemplo) para la bitacora.
	// Nunca se envia al cliente: puede tener detalles internos.
	Causa error
}

func (e *ErrorDominio) Error() string {
	if e.Causa != nil {
		return fmt.Sprintf("%s: %s: %v", e.Codigo, e.Mensaje, e.Causa)
	}
	return fmt.Sprintf("%s: %s", e.Codigo, e.Mensaje)
}

// Unwrap permite usar errors.Is y errors.As sobre la causa original.
func (e *ErrorDominio) Unwrap() error { return e.Causa }

// ConCausa adjunta el error original sin perder el codigo ni el mensaje.
func (e *ErrorDominio) ConCausa(causa error) *ErrorDominio {
	copia := *e
	copia.Causa = causa
	return &copia
}

// ConDetalles agrega la lista de problemas concretos, tipicamente los campos
// que no pasaron la validacion.
func (e *ErrorDominio) ConDetalles(detalles ...string) *ErrorDominio {
	copia := *e
	copia.Detalles = detalles
	return &copia
}

// Como extrae el ErrorDominio de un error envuelto. Si no lo es, devuelve false
// y quien llama debe tratarlo como un error interno.
func Como(err error) (*ErrorDominio, bool) {
	var dominio *ErrorDominio
	if errors.As(err, &dominio) {
		return dominio, true
	}
	return nil, false
}

// --- Constructores por familia de error ------------------------------------

// NoEncontrado: el recurso no existe, o existe pero es de otro usuario.
// Se responde igual en los dos casos para no revelar que el recurso existe.
func NoEncontrado(codigo, mensaje string) *ErrorDominio {
	return &ErrorDominio{Codigo: codigo, Mensaje: mensaje, HTTP: http.StatusNotFound}
}

// SolicitudInvalida: los datos que mando el cliente no sirven.
func SolicitudInvalida(codigo, mensaje string) *ErrorDominio {
	return &ErrorDominio{Codigo: codigo, Mensaje: mensaje, HTTP: http.StatusBadRequest}
}

// NoAutorizado: falta el token, esta vencido o es invalido.
func NoAutorizado(codigo, mensaje string) *ErrorDominio {
	return &ErrorDominio{Codigo: codigo, Mensaje: mensaje, HTTP: http.StatusUnauthorized}
}

// Prohibido: hay token valido pero no alcanza para esta operacion.
func Prohibido(codigo, mensaje string) *ErrorDominio {
	return &ErrorDominio{Codigo: codigo, Mensaje: mensaje, HTTP: http.StatusForbidden}
}

// Conflicto: choca con el estado actual (email repetido, cuenta con movimientos).
func Conflicto(codigo, mensaje string) *ErrorDominio {
	return &ErrorDominio{Codigo: codigo, Mensaje: mensaje, HTTP: http.StatusConflict}
}

// DemasiadasPeticiones: lo devuelve el limitador de /auth.
func DemasiadasPeticiones(codigo, mensaje string) *ErrorDominio {
	return &ErrorDominio{Codigo: codigo, Mensaje: mensaje, HTTP: http.StatusTooManyRequests}
}

// Interno: falla nuestra. El mensaje que ve el cliente es siempre generico.
func Interno(causa error) *ErrorDominio {
	return &ErrorDominio{
		Codigo:  CodigoErrorInterno,
		Mensaje: "Ocurrio un error inesperado. Intentalo de nuevo.",
		HTTP:    http.StatusInternalServerError,
		Causa:   causa,
	}
}
