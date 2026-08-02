package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// ProgresoMetas es la CONSULTA RELACIONAL 3.
//
// Objetivo: decir de cada meta de ahorro cuanto se lleva juntado, cuanto falta
// y que porcentaje del objetivo representa. Es lo que alimenta las barras de la
// pantalla de metas.
//
// Colecciones que cruza: metas -> aportaciones.
//
// Es la contraparte de la consulta 2: aquella compara lo presupuestado con lo
// gastado, esta compara lo que se quiere juntar con lo que ya se junto.
//
// La agregacion suma dinero y nada mas. Lo que depende de la fecha de hoy
// —estado, dias restantes y ritmo— lo calcula el servicio en Go, donde se puede
// probar con un reloj fijo (ver servicios/metas_ritmo.go).
//
// soloMeta es opcional: con el se pide el progreso de una sola meta (lo usa el
// detalle) y sin el, el de todas. Es la misma consulta en los dos casos, asi que
// el porcentaje no se puede calcular de dos maneras distintas.
//
// La version de mongosh y el resultado real contra la semilla estan en
// database/README.md.
func (r *Reportes) ProgresoMetas(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroMetas, soloMeta *bson.ObjectID) ([]modelos.ProgresoMeta, error) {
	// 1. Las metas de ESTE usuario.
	consulta := bson.M{"usuario_id": usuarioID}
	if soloMeta != nil {
		consulta["_id"] = *soloMeta
	} else if !filtro.IncluirArchivadas {
		// Al pedir una meta concreta se devuelve aunque este archivada: se llego
		// a ella por su id, no por el listado.
		consulta["archivada"] = false
	}

	etapas := []bson.M{
		{"$match": consulta},

		// 2. Las aportaciones de cada meta, ya sumadas.
		//
		//    Con let + pipeline y no con el $lookup simple: asi el $group ocurre
		//    DENTRO de la consulta relacionada y vuelve una sola fila por meta,
		//    en vez del arreglo completo de sus aportaciones.
		//
		//    El $eq de usuario_id no sobra aunque el $match ya haya filtrado las
		//    metas: sin el, el $lookup sumaria las aportaciones de cualquiera que
		//    apuntara a ese meta_id.
		{"$lookup": bson.M{
			"from": "aportaciones",
			"let":  bson.M{"meta": "$_id", "usr": "$usuario_id"},
			"pipeline": []bson.M{
				{"$match": bson.M{"$expr": bson.M{"$and": []bson.M{
					{"$eq": []any{"$usuario_id", "$$usr"}},
					{"$eq": []any{"$meta_id", "$$meta"}},
				}}}},
				{"$group": bson.M{
					"_id":         nil,
					"ahorrado":    bson.M{"$sum": "$monto"},
					"cantidad":    bson.M{"$sum": 1},
					"ultimaFecha": bson.M{"$max": "$fecha"},
				}},
			},
			"as": "resumen",
		}},

		// 3. Una meta sin aportaciones deja el arreglo vacio y $first da null.
		//    $ifNull lo convierte en 0 para que la meta igual aparezca, con su
		//    barra a cero. La fecha se deja en null a proposito: "nunca" no es
		//    una fecha y no hay que inventarsela.
		{"$addFields": bson.M{
			"ahorrado": bson.M{"$round": []any{
				bson.M{"$ifNull": []any{bson.M{"$first": "$resumen.ahorrado"}, 0}}, 2,
			}},
			"aportaciones": bson.M{"$ifNull": []any{bson.M{"$first": "$resumen.cantidad"}, 0}},
			"ultima_fecha": bson.M{"$first": "$resumen.ultimaFecha"},
		}},

		// 4. Lo que falta y que porcentaje se lleva.
		//
		//    restante se corta en 0 con $max: si se junto de mas, lo que falta es
		//    cero, no un numero negativo. Que se paso lo dice el porcentaje, que
		//    si puede superar el 100 y es la cifra que hay que ver.
		{"$addFields": bson.M{
			"restante": bson.M{"$round": []any{
				bson.M{"$max": []any{0, bson.M{"$subtract": []any{"$monto_objetivo", "$ahorrado"}}}}, 2,
			}},
			"porcentaje": bson.M{"$round": []any{
				bson.M{"$multiply": []any{
					bson.M{"$divide": []any{"$ahorrado", "$monto_objetivo"}}, 100,
				}}, 2,
			}},
		}},

		{"$project": bson.M{
			"_id":            0,
			"meta_id":        "$_id",
			"nombre":         1,
			"color":          1,
			"monto_objetivo": 1,
			"fecha_limite":   1,
			"notas":          1,
			"archivada":      1,
			"ahorrado":       1,
			"restante":       1,
			"porcentaje":     1,
			"aportaciones":   1,
			"ultima_fecha":   1,
		}},

		// 5. Lo que vence antes, primero.
		{"$sort": bson.M{"fecha_limite": 1}},
	}

	progreso := []modelos.ProgresoMeta{}
	if err := agregar(ctx, r.metas, etapas, &progreso, "al calcular el progreso de las metas"); err != nil {
		return nil, err
	}
	return progreso, nil
}
