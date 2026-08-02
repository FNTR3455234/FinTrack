package servicios

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// escenarioCSV monta un usuario con una cuenta, dos categorias y el servicio.
type escenarioCSV struct {
	servicio  *CSV
	repo      *transaccionesFalso
	usuarioID bson.ObjectID
	cuentaID  bson.ObjectID
	gastoID   bson.ObjectID
}

func nuevoEscenarioCSV(t *testing.T) escenarioCSV {
	t.Helper()

	repoTransacciones := nuevoTransaccionesFalso()
	repoCuentas := nuevoCuentasFalso()
	repoCategorias := nuevoCategoriasFalso()
	usuarioID := bson.NewObjectID()
	ctx := context.Background()

	cuenta := &modelos.Cuenta{UsuarioID: usuarioID, Nombre: "BBVA Debito"}
	require.NoError(t, repoCuentas.Crear(ctx, cuenta))

	gasto := &modelos.Categoria{UsuarioID: usuarioID, Nombre: "Supermercado", Tipo: modelos.TipoGasto}
	require.NoError(t, repoCategorias.Crear(ctx, gasto))

	ingreso := &modelos.Categoria{UsuarioID: usuarioID, Nombre: "Nomina", Tipo: modelos.TipoIngreso}
	require.NoError(t, repoCategorias.Crear(ctx, ingreso))

	return escenarioCSV{
		servicio:  NuevoCSV(repoTransacciones, repoCuentas, repoCategorias),
		repo:      repoTransacciones,
		usuarioID: usuarioID,
		cuentaID:  cuenta.ID,
		gastoID:   gasto.ID,
	}
}

// importar sube un archivo escrito a mano.
func (e escenarioCSV) importar(contenido string) (*modelos.ResultadoImportacion, error) {
	return e.servicio.Importar(context.Background(), e.usuarioID, strings.NewReader(contenido))
}

const encabezado = "fecha,tipo,cuenta,categoria,monto,descripcion,notas\n"

// --- exportacion ------------------------------------------------------------

func TestCSVExportar_EscribeElEncabezadoYUnaFilaPorMovimiento(t *testing.T) {
	e := nuevoEscenarioCSV(t)
	notas := "con cupon"
	require.NoError(t, e.repo.Crear(context.Background(), &modelos.Transaccion{
		UsuarioID: e.usuarioID, CuentaID: e.cuentaID, CategoriaID: e.gastoID,
		Tipo: modelos.TipoGasto, Monto: 850.5, Descripcion: "Despensa",
		Fecha: time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC), Notas: &notas,
	}))

	filas, err := e.servicio.Exportar(context.Background(), e.usuarioID, modelos.FiltroTransacciones{})

	require.NoError(t, err)
	require.Len(t, filas, 2)
	assert.Equal(t, modelos.ColumnasCSV, filas[0])
	assert.Equal(t, []string{
		"2026-07-03", "gasto", "BBVA Debito", "Supermercado", "850.50", "Despensa", "con cupon",
	}, filas[1], "la cuenta y la categoria salen por nombre, no por identificador")
}

func TestCSVExportar_UnMontoRedondoLlevaSusDosDecimales(t *testing.T) {
	e := nuevoEscenarioCSV(t)
	require.NoError(t, e.repo.Crear(context.Background(), &modelos.Transaccion{
		UsuarioID: e.usuarioID, CuentaID: e.cuentaID, CategoriaID: e.gastoID,
		Tipo: modelos.TipoGasto, Monto: 900, Descripcion: "Redondo",
		Fecha: time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
	}))

	filas, err := e.servicio.Exportar(context.Background(), e.usuarioID, modelos.FiltroTransacciones{})

	require.NoError(t, err)
	assert.Equal(t, "900.00", filas[1][4])
	assert.Equal(t, "", filas[1][6], "sin notas, la celda va vacia")
}

func TestCSVExportar_SinMovimientosDevuelveSoloElEncabezado(t *testing.T) {
	e := nuevoEscenarioCSV(t)

	filas, err := e.servicio.Exportar(context.Background(), e.usuarioID, modelos.FiltroTransacciones{})

	require.NoError(t, err)
	require.Len(t, filas, 1)
	assert.Equal(t, modelos.ColumnasCSV, filas[0])
}
