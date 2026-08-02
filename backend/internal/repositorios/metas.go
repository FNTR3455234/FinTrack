package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Metas accede a la coleccion metas.
type Metas struct {
	coleccion *mongo.Collection
}

// NuevoMetas construye el repositorio sobre la base indicada.
func NuevoMetas(bd *mongo.Database) *Metas {
	return &Metas{coleccion: bd.Collection("metas")}
}

// Crear inserta la meta y le asigna el _id que genero MongoDB.
func (r *Metas) Crear(ctx context.Context, meta *modelos.Meta) error {
	resultado, err := r.coleccion.InsertOne(ctx, meta)
	if err != nil {
		return traducir(err, "al insertar la meta")
	}
	if id, ok := resultado.InsertedID.(bson.ObjectID); ok {
		meta.ID = id
	}
	return nil
}

// Aqui NO hay un Listar.
//
// El listado de metas no sale de esta coleccion: sale de la agregacion
// ProgresoMetas (reportes_metas.go), que ademas trae lo ahorrado. Un Listar a
// secas devolveria metas sin progreso, que no le sirve a nadie, y tener los dos
// invitaria a llamar al equivocado. Lo delato la cobertura: estaba escrito y no
// lo usaba nadie.

// PorID busca una meta del usuario.
func (r *Metas) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Meta, error) {
	var meta modelos.Meta
	if err := r.coleccion.FindOne(ctx, suyoPorID(usuarioID, id)).Decode(&meta); err != nil {
		return nil, traducir(err, "al buscar la meta")
	}
	return &meta, nil
}

// Actualizar reemplaza los campos editables y devuelve la meta cambiada.
//
// creado_en no esta en la lista: es de cuando se creo y no cambia nunca.
func (r *Metas) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, m modelos.Meta) (*modelos.Meta, error) {
	cambios := bson.M{"$set": bson.M{
		"nombre":         m.Nombre,
		"monto_objetivo": m.MontoObjetivo,
		"fecha_limite":   m.FechaLimite,
		"color":          m.Color,
		"notas":          m.Notas,
		"archivada":      m.Archivada,
		"actualizado_en": m.ActualizadoEn,
	}}

	opciones := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var actualizada modelos.Meta
	err := r.coleccion.FindOneAndUpdate(ctx, suyoPorID(usuarioID, id), cambios, opciones).Decode(&actualizada)
	if err != nil {
		return nil, traducir(err, "al actualizar la meta")
	}
	return &actualizada, nil
}

// Eliminar borra la meta del usuario.
//
// Las aportaciones las borra antes el servicio: aqui solo se quita la meta.
func (r *Metas) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	resultado, err := r.coleccion.DeleteOne(ctx, suyoPorID(usuarioID, id))
	if err != nil {
		return traducir(err, "al eliminar la meta")
	}
	if resultado.DeletedCount == 0 {
		return ErrNoEncontrado
	}
	return nil
}
