package rutas

import (
	"net/http"
	"testing"
	"time"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Pruebas de las cinco agregaciones contra MongoDB de verdad, sobre un juego de
// datos chico y conocido: cada cifra que se afirma abajo se puede sumar a mano.

// finanzas es el escenario de los reportes.
type finanzas struct {
	api                     *api
	token                   string
	cuentaA, cuentaB        string
	supermercado, renta     string
	servicios, nomina       string
	presupuestoSupermercado modelos.Presupuesto
}

// escenarioDeReportes deja montado, para el usuario indicado:
//
//	cuentas:  A (saldo inicial 1000) y B (saldo inicial 1000)
//	julio 2026:  +20000 nomina | -1500 supermercado (dos gastos) | -2500 renta | -850 servicios
//	junio 2026:  -300 supermercado
//	presupuestos de julio:  supermercado 2000 | renta 2000 | servicios 1000
func escenarioDeReportes(t *testing.T, correo string) finanzas {
	t.Helper()

	a := routerReal(t)
	token := a.nuevoUsuario(correo)

	f := finanzas{
		api:          a,
		token:        token,
		cuentaA:      a.crearCuenta(token, "BBVA Debito"),
		cuentaB:      a.crearCuenta(token, "Efectivo"),
		supermercado: a.crearCategoria(token, "Supermercado", modelos.TipoGasto),
		renta:        a.crearCategoria(token, "Renta", modelos.TipoGasto),
		servicios:    a.crearCategoria(token, "Servicios", modelos.TipoGasto),
		nomina:       a.crearCategoria(token, "Nomina", modelos.TipoIngreso),
	}

	f.registrar(t, f.cuentaA, f.nomina, modelos.TipoIngreso, 20000, "Quincena", fechaUTC(2026, time.July, 1))
	f.registrar(t, f.cuentaA, f.supermercado, modelos.TipoGasto, 1000, "Despensa", fechaUTC(2026, time.July, 5))
	f.registrar(t, f.cuentaA, f.supermercado, modelos.TipoGasto, 500, "Despensa chica", fechaUTC(2026, time.July, 20))
	f.registrar(t, f.cuentaB, f.renta, modelos.TipoGasto, 2500, "Renta de julio", fechaUTC(2026, time.July, 2))
	f.registrar(t, f.cuentaA, f.servicios, modelos.TipoGasto, 850, "Luz y agua", fechaUTC(2026, time.July, 15))
	f.registrar(t, f.cuentaA, f.supermercado, modelos.TipoGasto, 300, "Despensa de junio", fechaUTC(2026, time.June, 18))

	f.presupuestoSupermercado = a.presupuestar(token, f.supermercado, 2000, 7, 2026)
	a.presupuestar(token, f.renta, 2000, 7, 2026)
	a.presupuestar(token, f.servicios, 1000, 7, 2026)

	return f
}

func (f finanzas) registrar(t *testing.T, cuenta, categoria, tipo string, monto float64, descripcion string, fecha time.Time) {
	t.Helper()
	f.api.datos(http.MethodPost, "/api/v1/transacciones", f.token,
		movimiento(cuenta, categoria, tipo, monto, descripcion, fecha), http.StatusCreated, nil)
}

// leer pide un reporte de julio de 2026 y deserializa la respuesta.
func (f finanzas) leer(reporte string, destino any) {
	f.api.datos(http.MethodGet, "/api/v1/reportes/"+reporte, f.token, nil, http.StatusOK, destino)
}
