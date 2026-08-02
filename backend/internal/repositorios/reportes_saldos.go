package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Saldos calcula cuanto dinero queda hoy en cada cuenta del usuario.
//
// El saldo no se guarda en la coleccion cuentas: se calcula sumando todos sus
// movimientos al saldo inicial. Guardarlo seria una segunda fuente de verdad
// que se desincroniza en cuanto una transaccion se edita o se borra y algo
// falla a mitad (ver docs/decisiones.md, decision 020).
//
// Cruza cuentas -> transacciones. El $lookup con let + pipeline agrupa los
// movimientos dentro de la propia consulta, asi que vuelve una sola fila por
// cuenta en vez del arreglo completo de sus transacciones.
func (r *Reportes) Saldos(ctx context.Context, usuarioID bson.ObjectID) ([]modelos.SaldoCuenta, error) {
	etapas := []bson.M{
		{"$match": bson.M{"usuario_id": usuarioID}},

		{"$lookup": bson.M{
			"from": "transacciones",
			"let":  bson.M{"cta": "$_id", "usr": "$usuario_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{"$expr": bson.M{"$and": []bson.M{
					{"$eq": []any{"$usuario_id", "$$usr"}},
					{"$eq": []any{"$cuenta_id", "$$cta"}},
				}}}},
				{"$group": bson.M{
					"_id":      nil,
					"ingresos": sumaPorTipo(modelos.TipoIngreso),
					"gastos":   sumaPorTipo(modelos.TipoGasto),
				}},
			},
			"as": "movimientos",
		}},

		// Una cuenta sin movimientos deja el arreglo vacio: sus sumas son 0 y su
		// saldo es el inicial.
		{"$addFields": bson.M{
			"ingresos": bson.M{"$ifNull": []any{bson.M{"$first": "$movimientos.ingresos"}, 0}},
			"gastos":   bson.M{"$ifNull": []any{bson.M{"$first": "$movimientos.gastos"}, 0}},
		}},

		{"$project": bson.M{
			"_id":           0,
			"cuenta_id":     "$_id",
			"nombre":        1,
			"tipo":          1,
			"color":         1,
			"archivada":     1,
			"saldo_inicial": 1,
			"ingresos":      bson.M{"$round": []any{"$ingresos", 2}},
			"gastos":        bson.M{"$round": []any{"$gastos", 2}},
			"saldo": bson.M{"$round": []any{
				bson.M{"$add": []any{
					"$saldo_inicial",
					bson.M{"$subtract": []any{"$ingresos", "$gastos"}},
				}}, 2,
			}},
		}},

		{"$sort": bson.D{{Key: "archivada", Value: 1}, {Key: "nombre", Value: 1}}},
	}

	saldos := []modelos.SaldoCuenta{}
	if err := agregar(ctx, r.cuentas, etapas, &saldos, "al calcular los saldos"); err != nil {
		return nil, err
	}
	return saldos, nil
}
