package handlers

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// ConfigurarValidador hace que los errores de validacion nombren el campo como
// se llama en el JSON (`token_refresco`) y no como se llama en Go
// (`TokenRefresco`). Se llama una vez al armar el router.
func ConfigurarValidador() {
	motor, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	motor.RegisterTagNameFunc(func(campo reflect.StructField) string {
		nombre := strings.SplitN(campo.Tag.Get("json"), ",", 2)[0]
		if nombre == "-" {
			return campo.Name
		}
		return nombre
	})
}

// enlazar lee el JSON del cuerpo, lo valida contra las etiquetas `binding` del
// DTO y, si algo falla, ya responde el error.
//
// Devuelve true solo si el handler puede seguir. El patron de uso es:
//
//	var peticion modelos.PeticionLogin
//	if !enlazar(c, &peticion) {
//	    return
//	}
func enlazar(c *gin.Context, destino any) bool {
	if err := c.ShouldBindJSON(destino); err != nil {
		respuestas.Fallo(c, errorDeEnlace(err))
		return false
	}
	return true
}

// errorDeEnlace distingue entre un JSON mal formado y un JSON correcto con
// datos que no cumplen las reglas.
func errorDeEnlace(err error) error {
	var fallosDeValidacion validator.ValidationErrors
	if errors.As(err, &fallosDeValidacion) {
		detalles := make([]string, 0, len(fallosDeValidacion))
		for _, fallo := range fallosDeValidacion {
			detalles = append(detalles, describir(fallo))
		}
		return fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoDatosInvalidos,
			"Algunos campos no son validos.").ConDetalles(detalles...)
	}

	return fintrackErrores.SolicitudInvalida(fintrackErrores.CodigoJSONInvalido,
		"El cuerpo de la peticion no es JSON valido.")
}

// describir traduce un fallo del validador a una frase en español.
// El nombre del campo es el del JSON, no el del struct de Go.
func describir(fallo validator.FieldError) string {
	campo := fallo.Field()

	switch fallo.Tag() {
	case "required":
		return campo + ": es obligatorio"
	case "email":
		return campo + ": debe ser un correo electronico valido"
	case "min":
		return fmt.Sprintf("%s: debe tener al menos %s caracteres", campo, fallo.Param())
	case "max":
		return fmt.Sprintf("%s: no puede pasar de %s caracteres", campo, fallo.Param())
	case "len":
		return fmt.Sprintf("%s: debe tener exactamente %s caracteres", campo, fallo.Param())
	case "alpha":
		return campo + ": solo puede tener letras"
	case "gt":
		return fmt.Sprintf("%s: debe ser mayor que %s", campo, fallo.Param())
	case "gte":
		return fmt.Sprintf("%s: debe ser mayor o igual que %s", campo, fallo.Param())
	case "lte":
		return fmt.Sprintf("%s: debe ser menor o igual que %s", campo, fallo.Param())
	case "oneof":
		return fmt.Sprintf("%s: debe ser uno de estos valores: %s", campo, fallo.Param())
	default:
		return fmt.Sprintf("%s: no cumple la regla %q", campo, fallo.Tag())
	}
}
