package rutas

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/respuestas"
)

// La comprobacion que pide la rubrica: dos usuarios de verdad, contra MongoDB,
// y ningun endpoint que devuelva o toque datos del otro.

func TestIntegracion_NingunEndpointDevuelveDatosDeOtroUsuario(t *testing.T) {
	a := routerReal(t)

	tokenAna := a.nuevoUsuario("ana@fintrack.mx")
	cuentaDeAna := a.crearCuenta(tokenAna, "Cuenta de Ana")
	categoriaDeAna := a.crearCategoria(tokenAna, "Categoria de Ana", modelos.TipoGasto)

	var movimientoDeAna modelos.Transaccion
	a.datos(http.MethodPost, "/api/v1/transacciones", tokenAna,
		movimiento(cuentaDeAna, categoriaDeAna, modelos.TipoGasto, 500, "Secreto de Ana", fechaUTC(2026, time.July, 5)),
		http.StatusCreated, &movimientoDeAna)

	tokenBeto := a.nuevoUsuario("beto@fintrack.mx")

	t.Run("los listados de Beto salen vacios", func(t *testing.T) {
		var cuentas []modelos.Cuenta
		a.datos(http.MethodGet, "/api/v1/cuentas", tokenBeto, nil, http.StatusOK, &cuentas)
		assert.Empty(t, cuentas)

		var categorias []modelos.Categoria
		a.datos(http.MethodGet, "/api/v1/categorias", tokenBeto, nil, http.StatusOK, &categorias)
		assert.Empty(t, categorias)

		var transacciones []modelos.Transaccion
		a.datos(http.MethodGet, "/api/v1/transacciones", tokenBeto, nil, http.StatusOK, &transacciones)
		assert.Empty(t, transacciones)
	})

	t.Run("leer por id responde 404, no 403", func(t *testing.T) {
		// Un 403 confirmaria que el recurso existe. Un 404 no dice nada.
		rutas := []string{
			"/api/v1/cuentas/" + cuentaDeAna,
			"/api/v1/categorias/" + categoriaDeAna,
			"/api/v1/transacciones/" + movimientoDeAna.ID.Hex(),
		}
		for _, ruta := range rutas {
			grabadora := a.llamar(http.MethodGet, ruta, tokenBeto, nil)
			assert.Equal(t, http.StatusNotFound, grabadora.Code, ruta)
			assert.NotContains(t, grabadora.Body.String(), "Secreto de Ana")
		}
	})

	t.Run("editar lo ajeno responde 404 y no cambia nada", func(t *testing.T) {
		saldo := 99999.0
		a.datos(http.MethodPut, "/api/v1/cuentas/"+cuentaDeAna, tokenBeto, modelos.PeticionCuenta{
			Nombre: "Robada", Tipo: modelos.CuentaDebito, SaldoInicial: &saldo, Color: "#000000",
		}, http.StatusNotFound, nil)

		var intacta modelos.Cuenta
		a.datos(http.MethodGet, "/api/v1/cuentas/"+cuentaDeAna, tokenAna, nil, http.StatusOK, &intacta)
		assert.Equal(t, "Cuenta de Ana", intacta.Nombre)
	})

	t.Run("borrar lo ajeno responde 404 y no borra nada", func(t *testing.T) {
		a.datos(http.MethodDelete, "/api/v1/transacciones/"+movimientoDeAna.ID.Hex(), tokenBeto, nil,
			http.StatusNotFound, nil)

		var sigueAhi modelos.Transaccion
		a.datos(http.MethodGet, "/api/v1/transacciones/"+movimientoDeAna.ID.Hex(), tokenAna, nil,
			http.StatusOK, &sigueAhi)
		assert.Equal(t, "Secreto de Ana", sigueAhi.Descripcion)
	})

	t.Run("no se puede colgar un movimiento de la cuenta de otro", func(t *testing.T) {
		// Beto manda en el cuerpo la cuenta y la categoria de Ana. Como las
		// comprobaciones filtran por usuario, para el simplemente no existen.
		grabadora := a.llamar(http.MethodPost, "/api/v1/transacciones", tokenBeto,
			movimiento(cuentaDeAna, categoriaDeAna, modelos.TipoGasto, 100, "Intruso", fechaUTC(2026, time.July, 6)))

		require.Equal(t, http.StatusNotFound, grabadora.Code)

		var cuerpo respuestas.SobreError
		require.NoError(t, json.Unmarshal(grabadora.Body.Bytes(), &cuerpo))
		assert.Equal(t, errores.CodigoCuentaNoEncontrada, cuerpo.Error.Codigo)
	})

	t.Run("el filtro por id ajeno no filtra nada de Ana", func(t *testing.T) {
		var transacciones []modelos.Transaccion
		a.datos(http.MethodGet, "/api/v1/transacciones?cuenta_id="+cuentaDeAna, tokenBeto, nil,
			http.StatusOK, &transacciones)
		assert.Empty(t, transacciones, "el usuario_id se aplica antes que cualquier filtro")
	})
}
