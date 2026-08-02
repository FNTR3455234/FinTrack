package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// GastosPorCategoria es la CONSULTA RELACIONAL 1.
//
// Objetivo: identificar en que categorias se concentra el gasto de un mes, con
// el total, el numero de movimientos y el peso porcentual de cada una sobre el
// gasto total del periodo. Responde a la pregunta "¿en que se me esta yendo el
// dinero?", que es la que ordena el tablero.
//
// Colecciones que cruza: transacciones -> categorias.
//
// La version de mongosh y el resultado real contra la semilla estan en
// database/README.md.
func (r *Reportes) GastosPorCategoria(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo) ([]modelos.GastoPorCategoria, error) {
	filtro := deUsuarioEnPeriodo(usuarioID, periodo)
	filtro["tipo"] = modelos.TipoGasto

	etapas := []bson.M{
		// 1. Solo los gastos de ESTE usuario dentro del mes. Va primero para que
		//    el indice (usuario_id, fecha) recorte el conjunto cuanto antes.
		{"$match": filtro},

		// 2. Se acumula por categoria: cuanto y cuantas veces.
		{"$group": bson.M{
			"_id":      "$categoria_id",
			"total":    bson.M{"$sum": "$monto"},
			"cantidad": bson.M{"$sum": 1},
		}},

		// 3. Se trae el nombre, el color y el icono de la categoria. Esta es la
		//    parte relacional: el nombre no esta en la transaccion, solo su id.
		{"$lookup": bson.M{
			"from":         "categorias",
			"localField":   "_id",
			"foreignField": "_id",
			"as":           "categoria",
		}},
		{"$unwind": "$categoria"},

		// 4. Suma de TODOS los grupos en un campo de cada documento, para poder
		//    sacar el porcentaje sin una segunda consulta ni un segundo recorrido
		//    en Go. La ventana "unbounded a unbounded" es la coleccion entera ya
		//    filtrada.
		{"$setWindowFields": bson.M{
			"output": bson.M{
				"gran_total": bson.M{
					"$sum":   "$total",
					"window": bson.M{"documents": []any{"unbounded", "unbounded"}},
				},
			},
		}},

		// 5. Se arma la fila final, ya redondeada a centavos.
		{"$project": bson.M{
			"_id":          0,
			"categoria_id": "$_id",
			"nombre":       "$categoria.nombre",
			"color":        "$categoria.color",
			"icono":        "$categoria.icono",
			"cantidad":     1,
			"total":        bson.M{"$round": []any{"$total", 2}},
			"porcentaje": bson.M{"$round": []any{
				bson.M{"$multiply": []any{
					bson.M{"$divide": []any{"$total", "$gran_total"}}, 100,
				}}, 2,
			}},
		}},

		// 6. De la categoria que mas pesa a la que menos, que es como se lee.
		{"$sort": bson.M{"total": -1}},
	}

	gastos := []modelos.GastoPorCategoria{}
	if err := agregar(ctx, r.transacciones, etapas, &gastos, "al calcular los gastos por categoria"); err != nil {
		return nil, err
	}
	return gastos, nil
}
