package servicios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// reportesFalso devuelve resultados fijos en lugar de ejecutar agregaciones.
//
// Aqui no se imita MongoDB: las agregaciones se prueban de verdad contra la
// base en internal/rutas. Lo que se prueba con este doble es lo que el servicio
// hace DESPUES: rellenar los meses vacios, sumar los saldos y contar el
// semaforo. Tambien guarda lo que se le pidio, para comprobar que el servicio
// calcula bien el rango.
type reportesFalso struct {
	gastos    []modelos.GastoPorCategoria
	estados   []modelos.EstadoPresupuesto
	totales   repositorios.TotalesDelMes
	tendencia []modelos.PuntoTendencia
	saldos    []modelos.SaldoCuenta

	desdePedido  modelos.Periodo
	hastaPedido  modelos.Periodo
	errorForzado error
}

func nuevoReportesFalso() *reportesFalso {
	return &reportesFalso{}
}

func (r *reportesFalso) GastosPorCategoria(_ context.Context, _ bson.ObjectID, _ modelos.Periodo) ([]modelos.GastoPorCategoria, error) {
	return r.gastos, r.errorForzado
}

func (r *reportesFalso) EstadoPresupuestos(_ context.Context, _ bson.ObjectID, _ modelos.Periodo, _ *bson.ObjectID) ([]modelos.EstadoPresupuesto, error) {
	return r.estados, r.errorForzado
}

func (r *reportesFalso) Totales(_ context.Context, _ bson.ObjectID, _ modelos.Periodo) (repositorios.TotalesDelMes, error) {
	return r.totales, r.errorForzado
}

func (r *reportesFalso) Tendencia(_ context.Context, _ bson.ObjectID, desde, hasta modelos.Periodo) ([]modelos.PuntoTendencia, error) {
	r.desdePedido, r.hastaPedido = desde, hasta
	return r.tendencia, r.errorForzado
}

func (r *reportesFalso) Saldos(_ context.Context, _ bson.ObjectID) ([]modelos.SaldoCuenta, error) {
	return r.saldos, r.errorForzado
}
