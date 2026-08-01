package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Cuentas accede a la coleccion cuentas.
type Cuentas struct {
	coleccion *mongo.Collection
}

// NuevoCuentas construye el repositorio sobre la base indicada.
func NuevoCuentas(bd *mongo.Database) *Cuentas {
	return &Cuentas{coleccion: bd.Collection("cuentas")}
}

// Crear inserta la cuenta y le asigna el _id que genero MongoDB.
func (r *Cuentas) Crear(ctx context.Context, cuenta *modelos.Cuenta) error {
	resultado, err := r.coleccion.InsertOne(ctx, cuenta)
	if err != nil {
		return traducir(err, "al insertar la cuenta")
	}
	if id, ok := resultado.InsertedID.(bson.ObjectID); ok {
		cuenta.ID = id
	}
	return nil
}

// Listar devuelve las cuentas del usuario, ordenadas por nombre.
// Si incluirArchivadas es false, deja fuera las archivadas.
func (r *Cuentas) Listar(ctx context.Context, usuarioID bson.ObjectID, incluirArchivadas bool) ([]modelos.Cuenta, error) {
	filtro := deUsuario(usuarioID)
	if !incluirArchivadas {
		filtro["archivada"] = false
	}

	cursor, err := r.coleccion.Find(ctx, filtro, options.Find().SetSort(bson.D{{Key: "nombre", Value: 1}}))
	if err != nil {
		return nil, traducir(err, "al listar las cuentas")
	}

	// Se inicializa vacio y no en nil para que el JSON sea [] y no null.
	cuentas := []modelos.Cuenta{}
	if err := cursor.All(ctx, &cuentas); err != nil {
		return nil, traducir(err, "al leer las cuentas")
	}
	return cuentas, nil
}

// PorID busca una cuenta del usuario. Si el id existe pero es de otro usuario,
// devuelve ErrNoEncontrado.
func (r *Cuentas) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Cuenta, error) {
	var cuenta modelos.Cuenta
	if err := r.coleccion.FindOne(ctx, suyoPorID(usuarioID, id)).Decode(&cuenta); err != nil {
		return nil, traducir(err, "al buscar la cuenta")
	}
	return &cuenta, nil
}

// Actualizar reemplaza los campos editables y devuelve la cuenta ya cambiada.
func (r *Cuentas) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, cuenta modelos.Cuenta) (*modelos.Cuenta, error) {
	cambios := bson.M{"$set": bson.M{
		"nombre":        cuenta.Nombre,
		"tipo":          cuenta.Tipo,
		"saldo_inicial": cuenta.SaldoInicial,
		"color":         cuenta.Color,
		"archivada":     cuenta.Archivada,
	}}

	opciones := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var actualizada modelos.Cuenta
	err := r.coleccion.FindOneAndUpdate(ctx, suyoPorID(usuarioID, id), cambios, opciones).Decode(&actualizada)
	if err != nil {
		return nil, traducir(err, "al actualizar la cuenta")
	}
	return &actualizada, nil
}

// Eliminar borra la cuenta del usuario.
func (r *Cuentas) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	resultado, err := r.coleccion.DeleteOne(ctx, suyoPorID(usuarioID, id))
	if err != nil {
		return traducir(err, "al eliminar la cuenta")
	}
	if resultado.DeletedCount == 0 {
		return ErrNoEncontrado
	}
	return nil
}

// Existe dice si la cuenta es del usuario. Lo usa el servicio de transacciones
// antes de aceptar una cuenta_id que llego en el cuerpo.
func (r *Cuentas) Existe(ctx context.Context, usuarioID, id bson.ObjectID) (bool, error) {
	total, err := r.coleccion.CountDocuments(ctx, suyoPorID(usuarioID, id), options.Count().SetLimit(1))
	if err != nil {
		return false, traducir(err, "al comprobar la cuenta")
	}
	return total > 0, nil
}
