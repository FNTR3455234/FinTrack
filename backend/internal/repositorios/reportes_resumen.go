package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// TotalesDelMes son las cifras crudas de un periodo, antes de mezclarlas con
// los saldos y los presupuestos.
type TotalesDelMes struct {
	Ingresos    float64 `bson:"ingresos"`
	Gastos      float64 `bson:"gastos"`
	Movimientos int     `bson:"movimientos"`
}

// sumaPorTipo suma el monto solo cuando la transaccion es del tipo indicado.
//
// Con un $cond por tipo se sacan las dos sumas en un solo recorrido; agrupar
// por tipo obligaria despues a buscar cual fila es cual en Go.
func sumaPorTipo(tipo string) bson.M {
	return bson.M{"$sum": bson.M{"$cond": []any{
		bson.M{"$eq": []any{"$tipo", tipo}}, "$monto", 0,
	}}}
}

// Totales devuelve los ingresos, los gastos y el numero de movimientos de un mes.
func (r *Reportes) Totales(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo) (TotalesDelMes, error) {
	etapas := []bson.M{
		{"$match": deUsuarioEnPeriodo(usuarioID, periodo)},
		{"$group": bson.M{
			"_id":         nil,
			"ingresos":    sumaPorTipo(modelos.TipoIngreso),
			"gastos":      sumaPorTipo(modelos.TipoGasto),
			"movimientos": bson.M{"$sum": 1},
		}},
		{"$project": bson.M{
			"_id":         0,
			"movimientos": 1,
			"ingresos":    bson.M{"$round": []any{"$ingresos", 2}},
			"gastos":      bson.M{"$round": []any{"$gastos", 2}},
		}},
	}

	filas := []TotalesDelMes{}
	if err := agregar(ctx, r.transacciones, etapas, &filas, "al calcular los totales del mes"); err != nil {
		return TotalesDelMes{}, err
	}
	// Un mes sin ningun movimiento no produce ninguna fila: son ceros.
	if len(filas) == 0 {
		return TotalesDelMes{}, nil
	}
	return filas[0], nil
}

// Tendencia devuelve un punto por cada mes en el que hubo movimientos, desde
// `desde` hasta el final de `hasta`.
//
// Los meses vacios NO salen de aqui: MongoDB solo agrupa lo que existe. Los
// rellena el servicio, que es donde se sabe que meses tenia que haber.
func (r *Reportes) Tendencia(ctx context.Context, usuarioID bson.ObjectID, desde, hasta modelos.Periodo) ([]modelos.PuntoTendencia, error) {
	inicio, _ := desde.Rango()
	_, fin := hasta.Rango()

	etapas := []bson.M{
		{"$match": bson.M{
			"usuario_id": usuarioID,
			"fecha":      bson.M{"$gte": inicio, "$lt": fin},
		}},

		// $year y $month leen la fecha en UTC, igual que Periodo.Rango: los dos
		// tienen que estar de acuerdo o un movimiento caeria en dos meses
		// distintos segun quien lo cuente.
		{"$group": bson.M{
			"_id": bson.M{
				"anio": bson.M{"$year": "$fecha"},
				"mes":  bson.M{"$month": "$fecha"},
			},
			"ingresos": sumaPorTipo(modelos.TipoIngreso),
			"gastos":   sumaPorTipo(modelos.TipoGasto),
			"cantidad": bson.M{"$sum": 1},
		}},

		{"$project": bson.M{
			"_id":      0,
			"cantidad": 1,
			"periodo":  bson.M{"mes": "$_id.mes", "anio": "$_id.anio"},
			"ingresos": bson.M{"$round": []any{"$ingresos", 2}},
			"gastos":   bson.M{"$round": []any{"$gastos", 2}},
		}},

		{"$sort": bson.D{{Key: "periodo.anio", Value: 1}, {Key: "periodo.mes", Value: 1}}},
	}

	puntos := []modelos.PuntoTendencia{}
	if err := agregar(ctx, r.transacciones, etapas, &puntos, "al calcular la tendencia"); err != nil {
		return nil, err
	}
	return puntos, nil
}
