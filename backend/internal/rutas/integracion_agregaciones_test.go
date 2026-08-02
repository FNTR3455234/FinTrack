package rutas

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// --- consulta relacional 1 --------------------------------------------------

func TestIntegracion_GastosPorCategoria(t *testing.T) {
	f := escenarioDeReportes(t, "gastos@fintrack.mx")

	var gastos []modelos.GastoPorCategoria
	f.leer("gastos-por-categoria?mes=7&anio=2026", &gastos)

	// El gasto total de julio es 4850. Junio no entra.
	require.Len(t, gastos, 3, "solo las categorias con gasto en el mes")

	assert.Equal(t, "Renta", gastos[0].Nombre, "ordenadas de mayor a menor gasto")
	assert.Equal(t, 2500.0, gastos[0].Total)
	assert.Equal(t, 1, gastos[0].Cantidad)
	assert.Equal(t, 51.55, gastos[0].Porcentaje) // 2500 / 4850

	assert.Equal(t, "Supermercado", gastos[1].Nombre)
	assert.Equal(t, 1500.0, gastos[1].Total, "los dos gastos de julio, sin el de junio")
	assert.Equal(t, 2, gastos[1].Cantidad)
	assert.Equal(t, 30.93, gastos[1].Porcentaje)

	assert.Equal(t, "Servicios", gastos[2].Nombre)
	assert.Equal(t, 850.0, gastos[2].Total)
	assert.Equal(t, 17.53, gastos[2].Porcentaje)

	// El $lookup trae los datos de la categoria, no solo su id.
	assert.NotEmpty(t, gastos[0].Color)
}

func TestIntegracion_GastosPorCategoriaDejaFueraLosIngresos(t *testing.T) {
	f := escenarioDeReportes(t, "gastos-ingresos@fintrack.mx")

	var gastos []modelos.GastoPorCategoria
	f.leer("gastos-por-categoria?mes=7&anio=2026", &gastos)

	for _, gasto := range gastos {
		assert.NotEqual(t, "Nomina", gasto.Nombre, "los 20000 de la nomina no son gasto")
	}
}

func TestIntegracion_GastosPorCategoriaDeUnMesVacio(t *testing.T) {
	f := escenarioDeReportes(t, "gastos-vacio@fintrack.mx")

	var gastos []modelos.GastoPorCategoria
	f.leer("gastos-por-categoria?mes=1&anio=2026", &gastos)

	assert.Empty(t, gastos, "un mes sin gastos es una lista vacia, no un error")
}

// --- consulta relacional 2 --------------------------------------------------

func TestIntegracion_EstadoDePresupuestos(t *testing.T) {
	f := escenarioDeReportes(t, "estado@fintrack.mx")

	var estados []modelos.EstadoPresupuesto
	f.leer("estado-presupuestos?mes=7&anio=2026", &estados)

	require.Len(t, estados, 3)

	// Renta: 2500 gastados de 2000 = 125 %.
	assert.Equal(t, "Renta", estados[0].Nombre, "ordenados por porcentaje, primero lo que apremia")
	assert.Equal(t, 2500.0, estados[0].Gastado)
	assert.Equal(t, -500.0, estados[0].Disponible, "el disponible se va a negativo")
	assert.Equal(t, 125.0, estados[0].PorcentajeUsado)
	assert.Equal(t, modelos.EstadoExcedido, estados[0].Estado)

	// Servicios: 850 de 1000 = 85 %.
	assert.Equal(t, "Servicios", estados[1].Nombre)
	assert.Equal(t, 85.0, estados[1].PorcentajeUsado)
	assert.Equal(t, modelos.EstadoAlerta, estados[1].Estado)

	// Supermercado: 1500 de 2000 = 75 %. El gasto de junio no cuenta.
	assert.Equal(t, "Supermercado", estados[2].Nombre)
	assert.Equal(t, 1500.0, estados[2].Gastado)
	assert.Equal(t, 500.0, estados[2].Disponible)
	assert.Equal(t, 75.0, estados[2].PorcentajeUsado)
	assert.Equal(t, modelos.EstadoOK, estados[2].Estado)

	assert.Equal(t, f.presupuestoSupermercado.ID, estados[2].PresupuestoID)
}

func TestIntegracion_UnPresupuestoSinGastosApareceEnCero(t *testing.T) {
	// Es el caso que justifica el $ifNull: sin el, el $lookup devuelve un
	// arreglo vacio y la categoria desapareceria del tablero justo cuando va
	// bien.
	f := escenarioDeReportes(t, "presupuesto-en-cero@fintrack.mx")
	f.api.presupuestar(f.token, f.supermercado, 3000, 9, 2026)

	var estados []modelos.EstadoPresupuesto
	f.leer("estado-presupuestos?mes=9&anio=2026", &estados)

	require.Len(t, estados, 1)
	assert.Zero(t, estados[0].Gastado)
	assert.Equal(t, 3000.0, estados[0].Disponible)
	assert.Zero(t, estados[0].PorcentajeUsado)
	assert.Equal(t, modelos.EstadoOK, estados[0].Estado)
}
