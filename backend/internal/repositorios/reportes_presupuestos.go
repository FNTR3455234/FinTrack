package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// EstadoPresupuestos es la CONSULTA RELACIONAL 2.
//
// Objetivo: comparar lo presupuestado contra lo realmente gastado en cada
// categoria durante un mes, y marcar cuales van en orden, cuales estan cerca
// del limite y cuales ya se pasaron. Es lo que alimenta las barras de color del
// tablero y las alertas al registrar un gasto.
//
// Colecciones que cruza: presupuestos -> categorias y presupuestos -> transacciones.
//
// soloCategoria es opcional: con el se pide el estado de una sola categoria
// (lo usa la alerta al registrar un gasto) y sin el, el del mes completo. Es la
// misma consulta en los dos casos, asi que el semaforo no se puede calcular de
// dos maneras distintas.
//
// La version de mongosh y el resultado real contra la semilla estan en
// database/README.md.
func (r *Reportes) EstadoPresupuestos(ctx context.Context, usuarioID bson.ObjectID, periodo modelos.Periodo, soloCategoria *bson.ObjectID) ([]modelos.EstadoPresupuesto, error) {
	inicio, fin := periodo.Rango()

	// 1. Los presupuestos de ESTE usuario para el mes pedido.
	filtro := bson.M{"usuario_id": usuarioID, "mes": periodo.Mes, "anio": periodo.Anio}
	if soloCategoria != nil {
		filtro["categoria_id"] = *soloCategoria
	}

	etapas := []bson.M{
		{"$match": filtro},

		// 2. Nombre y color de la categoria presupuestada.
		{"$lookup": bson.M{
			"from":         "categorias",
			"localField":   "categoria_id",
			"foreignField": "_id",
			"as":           "categoria",
		}},
		{"$unwind": "$categoria"},

		// 3. Lo realmente gastado en esa categoria durante el mes.
		//
		//    Aqui hace falta la forma con let + pipeline y no el $lookup simple:
		//    la condicion no es una igualdad de un campo, hay que cruzar por
		//    usuario Y categoria y ademas filtrar por tipo y por rango de fechas
		//    dentro de la coleccion relacionada.
		//
		//    El $eq de usuario_id no sobra aunque el $match ya filtre: sin el, el
		//    $lookup sumaria los gastos de cualquiera que use ese categoria_id.
		{"$lookup": bson.M{
			"from": "transacciones",
			"let":  bson.M{"cat": "$categoria_id", "usr": "$usuario_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{"$expr": bson.M{"$and": []bson.M{
					{"$eq": []any{"$usuario_id", "$$usr"}},
					{"$eq": []any{"$categoria_id", "$$cat"}},
					{"$eq": []any{"$tipo", modelos.TipoGasto}},
					{"$gte": []any{"$fecha", inicio}},
					{"$lt": []any{"$fecha", fin}},
				}}}},
				{"$group": bson.M{"_id": nil, "gastado": bson.M{"$sum": "$monto"}}},
			},
			"as": "movimientos",
		}},

		// 4. Si no hubo ni un gasto, el arreglo viene vacio y $first da null:
		//    $ifNull lo convierte en 0 para que la categoria igual aparezca.
		{"$addFields": bson.M{
			"gastado": bson.M{"$round": []any{
				bson.M{"$ifNull": []any{bson.M{"$first": "$movimientos.gastado"}, 0}}, 2,
			}},
		}},
		{"$addFields": bson.M{
			"disponible": bson.M{"$round": []any{
				bson.M{"$subtract": []any{"$monto_limite", "$gastado"}}, 2,
			}},
			"porcentaje_usado": bson.M{"$round": []any{
				bson.M{"$multiply": []any{
					bson.M{"$divide": []any{"$gastado", "$monto_limite"}}, 100,
				}}, 2,
			}},
		}},

		// 5. El semaforo. Los umbrales salen de las constantes de modelos para
		//    que el codigo y la consulta no se separen sin que se note.
		{"$addFields": bson.M{"estado": bson.M{"$switch": bson.M{
			"branches": []bson.M{
				{"case": bson.M{"$gt": []any{"$porcentaje_usado", modelos.UmbralExcedido}}, "then": modelos.EstadoExcedido},
				{"case": bson.M{"$gte": []any{"$porcentaje_usado", modelos.UmbralAlerta}}, "then": modelos.EstadoAlerta},
			},
			"default": modelos.EstadoOK,
		}}}},

		{"$project": bson.M{
			"_id":              0,
			"presupuesto_id":   "$_id",
			"categoria_id":     1,
			"nombre":           "$categoria.nombre",
			"color":            "$categoria.color",
			"monto_limite":     1,
			"gastado":          1,
			"disponible":       1,
			"porcentaje_usado": 1,
			"estado":           1,
		}},

		// 6. Primero lo que mas apremia.
		{"$sort": bson.M{"porcentaje_usado": -1}},
	}

	estados := []modelos.EstadoPresupuesto{}
	if err := agregar(ctx, r.presupuestos, etapas, &estados, "al calcular el estado de los presupuestos"); err != nil {
		return nil, err
	}
	return estados, nil
}

// EstadoDeCategoria devuelve como va el presupuesto de una sola categoria en un
// mes, o nil si esa categoria no tiene presupuesto ese mes.
func (r *Reportes) EstadoDeCategoria(ctx context.Context, usuarioID, categoriaID bson.ObjectID, periodo modelos.Periodo) (*modelos.EstadoPresupuesto, error) {
	estados, err := r.EstadoPresupuestos(ctx, usuarioID, periodo, &categoriaID)
	if err != nil {
		return nil, err
	}
	if len(estados) == 0 {
		return nil, nil
	}
	return &estados[0], nil
}
