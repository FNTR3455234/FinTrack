package servicios

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

func estado(nombre, semaforo string) modelos.EstadoPresupuesto {
	return modelos.EstadoPresupuesto{Nombre: nombre, Estado: semaforo}
}

func TestResumen_JuntaLosTotalesLosSaldosYElSemaforo(t *testing.T) {
	repo := nuevoReportesFalso()
	repo.totales = repositorios.TotalesDelMes{Ingresos: 33000, Gastos: 20311.20, Movimientos: 20}
	repo.saldos = []modelos.SaldoCuenta{
		{Nombre: "BBVA Debito", Saldo: 18500.40},
		{Nombre: "Efectivo", Saldo: 1200.10},
	}
	repo.estados = []modelos.EstadoPresupuesto{
		estado("Supermercado", modelos.EstadoExcedido),
		estado("Restaurantes", modelos.EstadoAlerta),
		estado("Servicios", modelos.EstadoAlerta),
		estado("Transporte", modelos.EstadoOK),
	}
	servicio := NuevoReportes(repo)

	resumen, err := servicio.Resumen(context.Background(), bson.NewObjectID(), julio2026())

	require.NoError(t, err)
	assert.Equal(t, julio2026(), resumen.Periodo)
	assert.Equal(t, 33000.0, resumen.Ingresos)
	assert.Equal(t, 20311.20, resumen.Gastos)
	assert.Equal(t, 12688.80, resumen.Balance)
	assert.Equal(t, 20, resumen.Movimientos)
	assert.Equal(t, 19700.50, resumen.SaldoTotal)
	assert.Equal(t, modelos.ContadorEstados{Total: 4, EnAlerta: 2, Excedidos: 1}, resumen.Presupuestos)
}

func TestResumen_UnMesSinNadaSonCerosYNoUnError(t *testing.T) {
	servicio := NuevoReportes(nuevoReportesFalso())

	resumen, err := servicio.Resumen(context.Background(), bson.NewObjectID(), julio2026())

	require.NoError(t, err)
	assert.Zero(t, resumen.Ingresos)
	assert.Zero(t, resumen.Gastos)
	assert.Zero(t, resumen.Balance)
	assert.Equal(t, modelos.ContadorEstados{}, resumen.Presupuestos)
}

func TestResumen_ElSaldoTotalDejaFueraLasCuentasArchivadas(t *testing.T) {
	repo := nuevoReportesFalso()
	repo.saldos = []modelos.SaldoCuenta{
		{Nombre: "BBVA Debito", Saldo: 10000},
		{Nombre: "Tarjeta vieja", Saldo: 7500, Archivada: true},
	}
	servicio := NuevoReportes(repo)

	resumen, err := servicio.Resumen(context.Background(), bson.NewObjectID(), julio2026())

	require.NoError(t, err)
	assert.Equal(t, 10000.0, resumen.SaldoTotal,
		"una cuenta archivada ya no es dinero disponible")
}

func TestResumen_ElBalanceNegativoNoSeRedondeaMal(t *testing.T) {
	repo := nuevoReportesFalso()
	repo.totales = repositorios.TotalesDelMes{Ingresos: 1000.10, Gastos: 1500.25}
	servicio := NuevoReportes(repo)

	resumen, err := servicio.Resumen(context.Background(), bson.NewObjectID(), julio2026())

	require.NoError(t, err)
	assert.Equal(t, -500.15, resumen.Balance)
}

func TestReportes_UnFalloDeLaBaseSaleComoErrorInterno(t *testing.T) {
	repo := nuevoReportesFalso()
	repo.errorForzado = errors.New("la base no respondio")
	servicio := NuevoReportes(repo)
	usuarioID := bson.NewObjectID()

	llamadas := map[string]func() error{
		"gastos por categoria": func() error {
			_, err := servicio.GastosPorCategoria(context.Background(), usuarioID, julio2026())
			return err
		},
		"estado de presupuestos": func() error {
			_, err := servicio.EstadoPresupuestos(context.Background(), usuarioID, julio2026())
			return err
		},
		"resumen": func() error {
			_, err := servicio.Resumen(context.Background(), usuarioID, julio2026())
			return err
		},
		"tendencia": func() error {
			_, err := servicio.Tendencia(context.Background(), usuarioID, julio2026(), 6)
			return err
		},
		"saldos": func() error {
			_, err := servicio.Saldos(context.Background(), usuarioID)
			return err
		},
	}

	for nombre, llamar := range llamadas {
		t.Run(nombre, func(t *testing.T) {
			dominio, ok := fintrackErrores.Como(llamar())
			require.True(t, ok)
			assert.Equal(t, fintrackErrores.CodigoErrorInterno, dominio.Codigo)
			assert.Equal(t, 500, dominio.HTTP)
		})
	}
}
