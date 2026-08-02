package servicios

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

func julio2026() modelos.Periodo { return modelos.Periodo{Mes: 7, Anio: 2026} }

func punto(mes, anio int, ingresos, gastos float64, cantidad int) modelos.PuntoTendencia {
	return modelos.PuntoTendencia{
		Periodo:  modelos.Periodo{Mes: mes, Anio: anio},
		Ingresos: ingresos,
		Gastos:   gastos,
		Cantidad: cantidad,
	}
}

func TestTendencia_DevuelveLosSeisMesesEnOrden(t *testing.T) {
	repo := nuevoReportesFalso()
	repo.tendencia = []modelos.PuntoTendencia{
		punto(2, 2026, 30000, 20000, 18),
		punto(7, 2026, 33000, 20311.20, 20),
	}
	servicio := NuevoReportes(repo)

	serie, err := servicio.Tendencia(context.Background(), bson.NewObjectID(), julio2026(), 6)

	require.NoError(t, err)
	require.Len(t, serie, 6)

	etiquetas := []string{}
	for _, p := range serie {
		etiquetas = append(etiquetas, p.Etiqueta)
	}
	assert.Equal(t, []string{"2026-02", "2026-03", "2026-04", "2026-05", "2026-06", "2026-07"}, etiquetas,
		"del mas viejo al mas reciente, terminando en el periodo pedido")
}

func TestTendencia_RellenaConCerosLosMesesSinMovimientos(t *testing.T) {
	repo := nuevoReportesFalso()
	repo.tendencia = []modelos.PuntoTendencia{punto(7, 2026, 33000, 20000, 20)}
	servicio := NuevoReportes(repo)

	serie, err := servicio.Tendencia(context.Background(), bson.NewObjectID(), julio2026(), 3)

	require.NoError(t, err)
	require.Len(t, serie, 3, "MongoDB solo agrupa lo que existe; los huecos los pone el servicio")

	assert.Equal(t, "2026-05", serie[0].Etiqueta)
	assert.Zero(t, serie[0].Ingresos)
	assert.Zero(t, serie[0].Gastos)
	assert.Zero(t, serie[0].Cantidad)
	assert.Equal(t, modelos.Periodo{Mes: 5, Anio: 2026}, serie[0].Periodo)

	assert.Equal(t, 33000.0, serie[2].Ingresos)
}

func TestTendencia_CalculaElBalanceDeCadaMes(t *testing.T) {
	repo := nuevoReportesFalso()
	repo.tendencia = []modelos.PuntoTendencia{punto(7, 2026, 33000, 20311.20, 20)}
	servicio := NuevoReportes(repo)

	serie, err := servicio.Tendencia(context.Background(), bson.NewObjectID(), julio2026(), 1)

	require.NoError(t, err)
	require.Len(t, serie, 1)
	assert.Equal(t, 12688.80, serie[0].Balance)
}

func TestTendencia_CruzaElCambioDeAño(t *testing.T) {
	repo := nuevoReportesFalso()
	servicio := NuevoReportes(repo)

	serie, err := servicio.Tendencia(context.Background(), bson.NewObjectID(),
		modelos.Periodo{Mes: 2, Anio: 2026}, 4)

	require.NoError(t, err)
	etiquetas := []string{}
	for _, p := range serie {
		etiquetas = append(etiquetas, p.Etiqueta)
	}
	assert.Equal(t, []string{"2025-11", "2025-12", "2026-01", "2026-02"}, etiquetas)
}

func TestTendencia_AjustaLosMesesFueraDeRango(t *testing.T) {
	casos := map[string]struct {
		pedidos  int
		esperado int
	}{
		"cero pasa al valor por defecto":     {0, modelos.MesesTendenciaPorDefecto},
		"negativo pasa al valor por defecto": {-3, modelos.MesesTendenciaPorDefecto},
		"pasado de largo se recorta":         {500, modelos.MesesTendenciaMaximo},
		"uno es valido":                      {1, 1},
	}

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio := NuevoReportes(nuevoReportesFalso())

			serie, err := servicio.Tendencia(context.Background(), bson.NewObjectID(), julio2026(), caso.pedidos)

			require.NoError(t, err)
			assert.Len(t, serie, caso.esperado)
		})
	}
}

func TestTendencia_PideAlRepositorioElRangoCompleto(t *testing.T) {
	repo := nuevoReportesFalso()
	servicio := NuevoReportes(repo)

	_, err := servicio.Tendencia(context.Background(), bson.NewObjectID(), julio2026(), 6)

	require.NoError(t, err)
	assert.Equal(t, modelos.Periodo{Mes: 2, Anio: 2026}, repo.desdePedido,
		"seis meses terminando en julio arrancan en febrero, no en enero")
	assert.Equal(t, julio2026(), repo.hastaPedido)
}
