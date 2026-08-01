package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Transacciones accede a la coleccion transacciones.
type Transacciones struct {
	coleccion *mongo.Collection
}

// NuevoTransacciones construye el repositorio sobre la base indicada.
func NuevoTransacciones(bd *mongo.Database) *Transacciones {
	return &Transacciones{coleccion: bd.Collection("transacciones")}
}

// Crear inserta la transaccion y le asigna el _id que genero MongoDB.
func (r *Transacciones) Crear(ctx context.Context, transaccion *modelos.Transaccion) error {
	resultado, err := r.coleccion.InsertOne(ctx, transaccion)
	if err != nil {
		return traducir(err, "al insertar la transaccion")
	}
	if id, ok := resultado.InsertedID.(bson.ObjectID); ok {
		transaccion.ID = id
	}
	return nil
}

// Listar devuelve una pagina de transacciones y el total que cumple el filtro.
//
// El total se necesita para la meta de paginacion, y sale de un CountDocuments
// aparte porque Find con Skip y Limit no sabe cuantos documentos hay en total.
func (r *Transacciones) Listar(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroTransacciones) ([]modelos.Transaccion, int64, error) {
	consulta := construirFiltro(usuarioID, filtro)

	total, err := r.coleccion.CountDocuments(ctx, consulta)
	if err != nil {
		return nil, 0, traducir(err, "al contar las transacciones")
	}

	opciones := options.Find().
		SetSort(construirOrden(filtro.Orden)).
		SetSkip(filtro.Saltar()).
		SetLimit(int64(filtro.Limite))

	cursor, err := r.coleccion.Find(ctx, consulta, opciones)
	if err != nil {
		return nil, 0, traducir(err, "al listar las transacciones")
	}

	transacciones := []modelos.Transaccion{}
	if err := cursor.All(ctx, &transacciones); err != nil {
		return nil, 0, traducir(err, "al leer las transacciones")
	}
	return transacciones, total, nil
}

// PorID busca una transaccion del usuario.
func (r *Transacciones) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Transaccion, error) {
	var transaccion modelos.Transaccion
	if err := r.coleccion.FindOne(ctx, suyoPorID(usuarioID, id)).Decode(&transaccion); err != nil {
		return nil, traducir(err, "al buscar la transaccion")
	}
	return &transaccion, nil
}

// Actualizar reemplaza los campos editables y devuelve la transaccion cambiada.
// creado_en no se toca: solo se actualiza actualizado_en.
func (r *Transacciones) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, t modelos.Transaccion) (*modelos.Transaccion, error) {
	cambios := bson.M{"$set": bson.M{
		"cuenta_id":      t.CuentaID,
		"categoria_id":   t.CategoriaID,
		"tipo":           t.Tipo,
		"monto":          t.Monto,
		"descripcion":    t.Descripcion,
		"fecha":          t.Fecha,
		"notas":          t.Notas,
		"actualizado_en": t.ActualizadoEn,
	}}

	opciones := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var actualizada modelos.Transaccion
	err := r.coleccion.FindOneAndUpdate(ctx, suyoPorID(usuarioID, id), cambios, opciones).Decode(&actualizada)
	if err != nil {
		return nil, traducir(err, "al actualizar la transaccion")
	}
	return &actualizada, nil
}

// Eliminar borra la transaccion del usuario.
func (r *Transacciones) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	resultado, err := r.coleccion.DeleteOne(ctx, suyoPorID(usuarioID, id))
	if err != nil {
		return traducir(err, "al eliminar la transaccion")
	}
	if resultado.DeletedCount == 0 {
		return ErrNoEncontrado
	}
	return nil
}

// ContarPorCuenta dice cuantas transacciones usan esa cuenta. Lo consulta el
// servicio antes de permitir borrarla, para no dejar movimientos huerfanos.
func (r *Transacciones) ContarPorCuenta(ctx context.Context, usuarioID, cuentaID bson.ObjectID) (int64, error) {
	filtro := deUsuario(usuarioID)
	filtro["cuenta_id"] = cuentaID

	total, err := r.coleccion.CountDocuments(ctx, filtro)
	if err != nil {
		return 0, traducir(err, "al contar las transacciones de la cuenta")
	}
	return total, nil
}

// ContarPorCategoria hace lo mismo para una categoria.
func (r *Transacciones) ContarPorCategoria(ctx context.Context, usuarioID, categoriaID bson.ObjectID) (int64, error) {
	filtro := deUsuario(usuarioID)
	filtro["categoria_id"] = categoriaID

	total, err := r.coleccion.CountDocuments(ctx, filtro)
	if err != nil {
		return 0, traducir(err, "al contar las transacciones de la categoria")
	}
	return total, nil
}
