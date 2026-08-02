package rutas

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// --- resumen, tendencia y saldos --------------------------------------------

func TestIntegracion_ResumenDelMes(t *testing.T) {
	f := escenarioDeReportes(t, "resumen@fintrack.mx")

	var resumen modelos.Resumen
	f.leer("resumen?mes=7&anio=2026", &resumen)

	assert.Equal(t, modelos.Periodo{Mes: 7, Anio: 2026}, resumen.Periodo)
	assert.Equal(t, 20000.0, resumen.Ingresos)
	assert.Equal(t, 4850.0, resumen.Gastos)
	assert.Equal(t, 15150.0, resumen.Balance)
	assert.Equal(t, 5, resumen.Movimientos, "los cinco de julio, sin el de junio")

	// Saldo: cuenta A = 1000 + 20000 - 2650 = 18350; cuenta B = 1000 - 2500 = -1500.
	assert.Equal(t, 16850.0, resumen.SaldoTotal, "el saldo es de siempre, no del mes")

	assert.Equal(t, modelos.ContadorEstados{Total: 3, EnAlerta: 1, Excedidos: 1}, resumen.Presupuestos)
}

func TestIntegracion_Tendencia(t *testing.T) {
	f := escenarioDeReportes(t, "tendencia@fintrack.mx")

	var serie []modelos.PuntoTendencia
	f.leer("tendencia?mes=7&anio=2026&meses=6", &serie)

	require.Len(t, serie, 6, "los seis meses aunque cuatro esten vacios")

	etiquetas := []string{}
	for _, punto := range serie {
		etiquetas = append(etiquetas, punto.Etiqueta)
	}
	assert.Equal(t, []string{"2026-02", "2026-03", "2026-04", "2026-05", "2026-06", "2026-07"}, etiquetas)

	// Febrero a mayo: sin nada.
	assert.Zero(t, serie[0].Ingresos)
	assert.Zero(t, serie[0].Gastos)
	assert.Zero(t, serie[0].Cantidad)

	// Junio: solo la despensa de 300.
	assert.Equal(t, 300.0, serie[4].Gastos)
	assert.Equal(t, -300.0, serie[4].Balance)
	assert.Equal(t, 1, serie[4].Cantidad)

	// Julio.
	assert.Equal(t, 20000.0, serie[5].Ingresos)
	assert.Equal(t, 4850.0, serie[5].Gastos)
	assert.Equal(t, 15150.0, serie[5].Balance)
	assert.Equal(t, 5, serie[5].Cantidad)
}

func TestIntegracion_Saldos(t *testing.T) {
	f := escenarioDeReportes(t, "saldos@fintrack.mx")

	var saldos []modelos.SaldoCuenta
	f.leer("saldos", &saldos)

	require.Len(t, saldos, 2)

	porNombre := map[string]modelos.SaldoCuenta{}
	for _, cuenta := range saldos {
		porNombre[cuenta.Nombre] = cuenta
	}

	// A: 1000 inicial + 20000 de ingresos - 2650 de gastos.
	a := porNombre["BBVA Debito"]
	assert.Equal(t, 20000.0, a.Ingresos)
	assert.Equal(t, 2650.0, a.Gastos)
	assert.Equal(t, 18350.0, a.Saldo)

	// B: 1000 inicial - 2500 de renta. Puede quedar en negativo.
	b := porNombre["Efectivo"]
	assert.Equal(t, -1500.0, b.Saldo)
}

func TestIntegracion_UnaCuentaSinMovimientosConservaSuSaldoInicial(t *testing.T) {
	a := routerReal(t)
	token := a.nuevoUsuario("cuenta-nueva@fintrack.mx")
	a.crearCuenta(token, "Recien abierta")

	var saldos []modelos.SaldoCuenta
	a.datos(http.MethodGet, "/api/v1/reportes/saldos", token, nil, http.StatusOK, &saldos)

	require.Len(t, saldos, 1)
	assert.Equal(t, 1000.0, saldos[0].Saldo, "sin movimientos, el saldo es el inicial")
	assert.Zero(t, saldos[0].Ingresos)
	assert.Zero(t, saldos[0].Gastos)
}

// --- aislamiento ------------------------------------------------------------

func TestIntegracion_LosReportesNuncaMezclanUsuarios(t *testing.T) {
	// La prueba mas importante de la fase: las agregaciones cruzan colecciones,
	// y un $lookup al que se le olvide el usuario_id sumaria dinero ajeno sin
	// que ningun 403 lo delate.
	f := escenarioDeReportes(t, "ana-reportes@fintrack.mx")
	a := f.api

	beto := a.nuevoUsuario("beto-reportes@fintrack.mx")
	cuentaBeto := a.crearCuenta(beto, "Cuenta de Beto")
	categoriaBeto := a.crearCategoria(beto, "Supermercado", modelos.TipoGasto)
	a.presupuestar(beto, categoriaBeto, 500, 7, 2026)
	a.datos(http.MethodPost, "/api/v1/transacciones", beto,
		movimiento(cuentaBeto, categoriaBeto, modelos.TipoGasto, 99999, "Gasto enorme", fechaUTC(2026, time.July, 9)),
		http.StatusCreated, nil)

	// Los reportes de Ana no se enteran de los 99999 de Beto.
	var gastos []modelos.GastoPorCategoria
	f.leer("gastos-por-categoria?mes=7&anio=2026", &gastos)
	for _, gasto := range gastos {
		assert.Less(t, gasto.Total, 3000.0, "ninguna categoria de Ana llega a 3000")
	}

	var estados []modelos.EstadoPresupuesto
	f.leer("estado-presupuestos?mes=7&anio=2026", &estados)
	require.Len(t, estados, 3, "los tres de Ana, ninguno de Beto")
	for _, estado := range estados {
		assert.Less(t, estado.Gastado, 3000.0)
	}

	var resumen modelos.Resumen
	f.leer("resumen?mes=7&anio=2026", &resumen)
	assert.Equal(t, 4850.0, resumen.Gastos)

	var saldos []modelos.SaldoCuenta
	f.leer("saldos", &saldos)
	assert.Len(t, saldos, 2, "solo las dos cuentas de Ana")

	// Y del otro lado, Beto ve lo suyo y solo lo suyo.
	var deBeto []modelos.EstadoPresupuesto
	a.datos(http.MethodGet, "/api/v1/reportes/estado-presupuestos?mes=7&anio=2026", beto, nil, http.StatusOK, &deBeto)
	require.Len(t, deBeto, 1)
	assert.Equal(t, 99999.0, deBeto[0].Gastado)
	assert.Equal(t, modelos.EstadoExcedido, deBeto[0].Estado)
}

func TestIntegracion_LosReportesExigenToken(t *testing.T) {
	a := routerReal(t)

	rutas := []string{
		"/api/v1/reportes/gastos-por-categoria",
		"/api/v1/reportes/estado-presupuestos",
		"/api/v1/reportes/resumen",
		"/api/v1/reportes/tendencia",
		"/api/v1/reportes/saldos",
		"/api/v1/presupuestos",
	}

	for _, ruta := range rutas {
		t.Run(ruta, func(t *testing.T) {
			a.datos(http.MethodGet, ruta, "", nil, http.StatusUnauthorized, nil)
		})
	}
}
