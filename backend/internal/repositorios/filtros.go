package repositorios

import (
	"regexp"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// construirFiltro traduce los criterios del listado al filtro de MongoDB.
//
// El usuario_id se pone SIEMPRE y primero: aunque el cliente mande cualquier
// combinacion de filtros, nunca puede salirse de sus propios documentos.
func construirFiltro(usuarioID bson.ObjectID, f modelos.FiltroTransacciones) bson.M {
	consulta := deUsuario(usuarioID)

	// Rango de fechas. Se usa $gte / $lte para que los extremos entren.
	rango := bson.M{}
	if f.Desde != nil {
		rango["$gte"] = *f.Desde
	}
	if f.Hasta != nil {
		rango["$lte"] = *f.Hasta
	}
	if len(rango) > 0 {
		consulta["fecha"] = rango
	}

	if f.Tipo != "" {
		consulta["tipo"] = f.Tipo
	}
	if f.CategoriaID != nil {
		consulta["categoria_id"] = *f.CategoriaID
	}
	if f.CuentaID != nil {
		consulta["cuenta_id"] = *f.CuentaID
	}

	if f.Busqueda != "" {
		// QuoteMeta escapa los caracteres especiales de la expresion regular.
		// Sin eso, una busqueda como ".*" recorreria toda la coleccion y una
		// mal formada haria fallar la consulta.
		patron := regexp.QuoteMeta(f.Busqueda)
		consulta["$or"] = []bson.M{
			{"descripcion": bson.M{"$regex": patron, "$options": "i"}},
			{"notas": bson.M{"$regex": patron, "$options": "i"}},
		}
	}

	return consulta
}

// construirOrden traduce el parametro orden al criterio de MongoDB.
//
// Siempre se desempata por _id: sin eso, dos transacciones con la misma fecha
// pueden salir en distinto orden en cada consulta y repetirse o perderse entre
// paginas.
func construirOrden(orden string) bson.D {
	switch orden {
	case modelos.OrdenFechaAsc:
		return bson.D{{Key: "fecha", Value: 1}, {Key: "_id", Value: 1}}
	case modelos.OrdenMontoDesc:
		return bson.D{{Key: "monto", Value: -1}, {Key: "_id", Value: 1}}
	case modelos.OrdenMontoAsc:
		return bson.D{{Key: "monto", Value: 1}, {Key: "_id", Value: 1}}
	default:
		// El indice (usuario_id, fecha DESC) sirve justo para este caso, que
		// es el listado que ve el usuario al abrir la pantalla.
		return bson.D{{Key: "fecha", Value: -1}, {Key: "_id", Value: -1}}
	}
}
