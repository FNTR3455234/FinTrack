package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Categorias accede a la coleccion categorias.
type Categorias struct {
	coleccion *mongo.Collection
}

// NuevoCategorias construye el repositorio sobre la base indicada.
func NuevoCategorias(bd *mongo.Database) *Categorias {
	return &Categorias{coleccion: bd.Collection("categorias")}
}

// Crear inserta la categoria y le asigna el _id que genero MongoDB.
func (r *Categorias) Crear(ctx context.Context, categoria *modelos.Categoria) error {
	resultado, err := r.coleccion.InsertOne(ctx, categoria)
	if err != nil {
		return traducir(err, "al insertar la categoria")
	}
	if id, ok := resultado.InsertedID.(bson.ObjectID); ok {
		categoria.ID = id
	}
	return nil
}

// Listar devuelve las categorias del usuario, ordenadas por tipo y nombre.
// Si tipo no viene vacio, filtra tambien por ingreso o gasto.
func (r *Categorias) Listar(ctx context.Context, usuarioID bson.ObjectID, tipo string, incluirArchivadas bool) ([]modelos.Categoria, error) {
	filtro := deUsuario(usuarioID)
	if !incluirArchivadas {
		filtro["archivada"] = false
	}
	if tipo != "" {
		filtro["tipo"] = tipo
	}

	orden := options.Find().SetSort(bson.D{{Key: "tipo", Value: 1}, {Key: "nombre", Value: 1}})
	cursor, err := r.coleccion.Find(ctx, filtro, orden)
	if err != nil {
		return nil, traducir(err, "al listar las categorias")
	}

	categorias := []modelos.Categoria{}
	if err := cursor.All(ctx, &categorias); err != nil {
		return nil, traducir(err, "al leer las categorias")
	}
	return categorias, nil
}

// PorID busca una categoria del usuario.
func (r *Categorias) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Categoria, error) {
	var categoria modelos.Categoria
	if err := r.coleccion.FindOne(ctx, suyoPorID(usuarioID, id)).Decode(&categoria); err != nil {
		return nil, traducir(err, "al buscar la categoria")
	}
	return &categoria, nil
}

// Actualizar reemplaza los campos editables y devuelve la categoria cambiada.
func (r *Categorias) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, categoria modelos.Categoria) (*modelos.Categoria, error) {
	cambios := bson.M{"$set": bson.M{
		"nombre":    categoria.Nombre,
		"tipo":      categoria.Tipo,
		"color":     categoria.Color,
		"icono":     categoria.Icono,
		"archivada": categoria.Archivada,
	}}

	opciones := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var actualizada modelos.Categoria
	err := r.coleccion.FindOneAndUpdate(ctx, suyoPorID(usuarioID, id), cambios, opciones).Decode(&actualizada)
	if err != nil {
		return nil, traducir(err, "al actualizar la categoria")
	}
	return &actualizada, nil
}

// Eliminar borra la categoria del usuario.
func (r *Categorias) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	resultado, err := r.coleccion.DeleteOne(ctx, suyoPorID(usuarioID, id))
	if err != nil {
		return traducir(err, "al eliminar la categoria")
	}
	if resultado.DeletedCount == 0 {
		return ErrNoEncontrado
	}
	return nil
}
