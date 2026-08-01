package errores

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructores_AsignanElCodigoHTTPCorrecto(t *testing.T) {
	casos := []struct {
		nombre   string
		err      *ErrorDominio
		esperado int
	}{
		{"no encontrado", NoEncontrado(CodigoCuentaNoEncontrada, "x"), http.StatusNotFound},
		{"solicitud invalida", SolicitudInvalida(CodigoDatosInvalidos, "x"), http.StatusBadRequest},
		{"no autorizado", NoAutorizado(CodigoTokenVencido, "x"), http.StatusUnauthorized},
		{"prohibido", Prohibido("X", "x"), http.StatusForbidden},
		{"conflicto", Conflicto(CodigoEmailYaRegistrado, "x"), http.StatusConflict},
		{"demasiadas peticiones", DemasiadasPeticiones(CodigoDemasiadosIntentos, "x"), http.StatusTooManyRequests},
		{"interno", Interno(nil), http.StatusInternalServerError},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			assert.Equal(t, caso.esperado, caso.err.HTTP)
			assert.NotEmpty(t, caso.err.Codigo)
			assert.NotEmpty(t, caso.err.Mensaje)
		})
	}
}

func TestError_IncluyeLaCausaCuandoLaHay(t *testing.T) {
	sinCausa := NoEncontrado(CodigoCuentaNoEncontrada, "La cuenta no existe.")
	assert.Equal(t, "CUENTA_NO_ENCONTRADA: La cuenta no existe.", sinCausa.Error())

	conCausa := sinCausa.ConCausa(errors.New("mongo: no documents"))
	assert.Contains(t, conCausa.Error(), "mongo: no documents")
}

func TestConCausa_NoModificaElErrorOriginal(t *testing.T) {
	// Los constructores devuelven valores reutilizables; si ConCausa mutara el
	// original, dos peticiones distintas compartirian la causa.
	original := NoEncontrado(CodigoCuentaNoEncontrada, "La cuenta no existe.")

	copia := original.ConCausa(errors.New("falla de red"))

	assert.Nil(t, original.Causa)
	assert.NotNil(t, copia.Causa)
	assert.Equal(t, original.Codigo, copia.Codigo)
}

func TestConDetalles_NoModificaElErrorOriginal(t *testing.T) {
	original := SolicitudInvalida(CodigoDatosInvalidos, "Revisa los campos.")

	copia := original.ConDetalles("monto debe ser mayor que 0", "fecha es obligatoria")

	assert.Empty(t, original.Detalles)
	assert.Len(t, copia.Detalles, 2)
}

func TestComo_EncuentraElErrorDominioAunqueEsteEnvuelto(t *testing.T) {
	original := Conflicto(CodigoEmailYaRegistrado, "Ese correo ya esta registrado.")
	envuelto := fmt.Errorf("al registrar el usuario: %w", original)

	dominio, ok := Como(envuelto)

	require.True(t, ok)
	assert.Equal(t, CodigoEmailYaRegistrado, dominio.Codigo)
	assert.Equal(t, http.StatusConflict, dominio.HTTP)
}

func TestComo_DevuelveFalsoConUnErrorCualquiera(t *testing.T) {
	dominio, ok := Como(errors.New("se cayo la red"))

	assert.False(t, ok)
	assert.Nil(t, dominio)
}

func TestUnwrap_PermiteUsarErrorsIsSobreLaCausa(t *testing.T) {
	causaRaiz := errors.New("conexion rechazada")
	err := Interno(causaRaiz)

	assert.True(t, errors.Is(err, causaRaiz))
}
