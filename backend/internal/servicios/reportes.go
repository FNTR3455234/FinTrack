package servicios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// RepositorioReportes es lo que el servicio necesita de la capa de datos: las
// cinco agregaciones.
type RepositorioReportes interface {
	GastosPorCategoria(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo) ([]modelos.GastoPorCategoria, error)
	EstadoPresupuestos(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo, soloCategoria *bson.ObjectID) ([]modelos.EstadoPresupuesto, error)
	Totales(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo) (repositorios.TotalesDelMes, error)
	Tendencia(ctx context.Context, usuarioID bson.ObjectID, desde, hasta modelos.Periodo) ([]modelos.PuntoTendencia, error)
	Saldos(ctx context.Context, usuarioID bson.ObjectID) ([]modelos.SaldoCuenta, error)
}

// Reportes resuelve las consultas de analisis.
//
// Aqui no hay estado ni reglas de escritura: el servicio pide la agregacion,
// completa lo que MongoDB no puede saber (los meses vacios) y traduce errores.
type Reportes struct {
	repositorio RepositorioReportes
}

// NuevoReportes arma el servicio con su repositorio.
func NuevoReportes(r RepositorioReportes) *Reportes {
	return &Reportes{repositorio: r}
}

// GastosPorCategoria devuelve en que se fue el dinero durante un mes.
func (s *Reportes) GastosPorCategoria(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo) ([]modelos.GastoPorCategoria, error) {
	gastos, err := s.repositorio.GastosPorCategoria(ctx, usuarioID, periodo)
	if err != nil {
		return nil, errores.Interno(err)
	}
	return gastos, nil
}

// EstadoPresupuestos devuelve como va cada presupuesto del mes.
func (s *Reportes) EstadoPresupuestos(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo) ([]modelos.EstadoPresupuesto, error) {
	estados, err := s.repositorio.EstadoPresupuestos(ctx, usuarioID, periodo, nil)
	if err != nil {
		return nil, errores.Interno(err)
	}
	return estados, nil
}

// Saldos devuelve cuanto queda hoy en cada cuenta.
func (s *Reportes) Saldos(ctx context.Context, usuarioID bson.ObjectID) ([]modelos.SaldoCuenta, error) {
	saldos, err := s.repositorio.Saldos(ctx, usuarioID)
	if err != nil {
		return nil, errores.Interno(err)
	}
	return saldos, nil
}

// Resumen junta en un solo objeto las cifras que encabezan el tablero: los
// totales del mes, el dinero disponible y como va el semaforo de presupuestos.
//
// Son tres agregaciones distintas y se lanzan en orden, no en paralelo: son
// consultas de milisegundos sobre indices y el codigo secuencial se lee y se
// depura mucho mejor que tres goroutines con su manejo de errores.
func (s *Reportes) Resumen(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo) (*modelos.Resumen, error) {
	totales, err := s.repositorio.Totales(ctx, usuarioID, periodo)
	if err != nil {
		return nil, errores.Interno(err)
	}

	saldos, err := s.repositorio.Saldos(ctx, usuarioID)
	if err != nil {
		return nil, errores.Interno(err)
	}

	estados, err := s.repositorio.EstadoPresupuestos(ctx, usuarioID, periodo, nil)
	if err != nil {
		return nil, errores.Interno(err)
	}

	return &modelos.Resumen{
		Periodo:      periodo,
		Ingresos:     totales.Ingresos,
		Gastos:       totales.Gastos,
		Balance:      redondear(totales.Ingresos - totales.Gastos),
		Movimientos:  totales.Movimientos,
		SaldoTotal:   saldoDisponible(saldos),
		Presupuestos: contarEstados(estados),
	}, nil
}

// saldoDisponible suma el saldo de las cuentas que siguen en uso.
//
// Las archivadas quedan fuera a proposito: se archivan justo cuando ya no
// tienen dinero o se dejaron de usar, y sumarlas inflaria el disponible.
func saldoDisponible(saldos []modelos.SaldoCuenta) float64 {
	var total float64
	for _, cuenta := range saldos {
		if !cuenta.Archivada {
			total += cuenta.Saldo
		}
	}
	return redondear(total)
}

// contarEstados reduce el semaforo del mes a tres numeros.
func contarEstados(estados []modelos.EstadoPresupuesto) modelos.ContadorEstados {
	contador := modelos.ContadorEstados{Total: len(estados)}
	for _, estado := range estados {
		switch estado.Estado {
		case modelos.EstadoAlerta:
			contador.EnAlerta++
		case modelos.EstadoExcedido:
			contador.Excedidos++
		}
	}
	return contador
}
