package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Aportaciones accede a la coleccion aportaciones.
type Aportaciones struct {
	coleccion *mongo.Collection
}

// NuevoAportaciones construye el repositorio sobre la base indicada.
func NuevoAportaciones(bd *mongo.Database) *Aportaciones {
	return &Aportaciones{coleccion: bd.Collection("aportaciones")}
}

// Crear inserta la aportacion y le asigna el _id que genero MongoDB.
func (r *Aportaciones) Crear(ctx context.Context, aportacion *modelos.Aportacion) error {
	resultado, err := r.coleccion.InsertOne(ctx, aportacion)
	if err != nil {
		return traducir(err, "al insertar la aportacion")
	}
	if id, ok := resultado.InsertedID.(bson.ObjectID); ok {
		aportacion.ID = id
	}
	return nil
}

// DeMeta devuelve las aportaciones de una meta, de la mas reciente a la mas
// vieja, con un tope de seguridad.
func (r *Aportaciones) DeMeta(ctx context.Context, usuarioID, metaID bson.ObjectID) ([]modelos.Aportacion, error) {
	filtro := deUsuario(usuarioID)
	filtro["meta_id"] = metaID

	opciones := options.Find().
		SetSort(bson.D{{Key: "fecha", Value: -1}}).
		SetLimit(modelos.MaximoAportacionesPorMeta)

	cursor, err := r.coleccion.Find(ctx, filtro, opciones)
	if err != nil {
		return nil, traducir(err, "al listar las aportaciones")
	}

	aportaciones := []modelos.Aportacion{}
	if err := cursor.All(ctx, &aportaciones); err != nil {
		return nil, traducir(err, "al leer las aportaciones")
	}
	return aportaciones, nil
}

// Eliminar borra una aportacion concreta de una meta del usuario.
//
// El filtro lleva las tres llaves —aportacion, meta y usuario— y no solo el
// _id: asi una aportacion de otra meta no se puede borrar pasando su id por una
// ruta que no le corresponde.
func (r *Aportaciones) Eliminar(ctx context.Context, usuarioID, metaID, id bson.ObjectID) error {
	filtro := suyoPorID(usuarioID, id)
	filtro["meta_id"] = metaID

	resultado, err := r.coleccion.DeleteOne(ctx, filtro)
	if err != nil {
		return traducir(err, "al eliminar la aportacion")
	}
	if resultado.DeletedCount == 0 {
		return ErrNoEncontrado
	}
	return nil
}

// EliminarDeMeta borra todas las aportaciones de una meta. Lo llama el servicio
// ANTES de borrar la meta.
//
// No devuelve error si no habia ninguna: borrar una meta sin aportaciones es
// perfectamente normal.
func (r *Aportaciones) EliminarDeMeta(ctx context.Context, usuarioID, metaID bson.ObjectID) (int64, error) {
	filtro := deUsuario(usuarioID)
	filtro["meta_id"] = metaID

	resultado, err := r.coleccion.DeleteMany(ctx, filtro)
	if err != nil {
		return 0, traducir(err, "al eliminar las aportaciones de la meta")
	}
	return resultado.DeletedCount, nil
}
