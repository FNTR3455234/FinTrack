package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Presupuestos accede a la coleccion presupuestos.
type Presupuestos struct {
	coleccion *mongo.Collection
}

// NuevoPresupuestos construye el repositorio sobre la base indicada.
func NuevoPresupuestos(bd *mongo.Database) *Presupuestos {
	return &Presupuestos{coleccion: bd.Collection("presupuestos")}
}

// Crear inserta el presupuesto y le asigna el _id que genero MongoDB.
//
// Si ya existe uno para esa categoria y periodo, el indice unico hace fallar el
// insert y traducir lo convierte en ErrDuplicado. No se consulta antes de
// insertar: entre la consulta y el insert cabe otra peticion, y quien decide de
// verdad es el indice (ver docs/decisiones.md, decision 013).
func (r *Presupuestos) Crear(ctx context.Context, presupuesto *modelos.Presupuesto) error {
	resultado, err := r.coleccion.InsertOne(ctx, presupuesto)
	if err != nil {
		return traducir(err, "al insertar el presupuesto")
	}
	if id, ok := resultado.InsertedID.(bson.ObjectID); ok {
		presupuesto.ID = id
	}
	return nil
}

// Listar devuelve los presupuestos del usuario, opcionalmente los de un periodo.
func (r *Presupuestos) Listar(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroPresupuestos) ([]modelos.Presupuesto, error) {
	consulta := deUsuario(usuarioID)
	if filtro.Periodo != nil {
		consulta["mes"] = filtro.Periodo.Mes
		consulta["anio"] = filtro.Periodo.Anio
	}

	// Del mes mas reciente al mas viejo, y dentro del mes por monto: el limite
	// mas grande es el que manda en el presupuesto.
	orden := options.Find().SetSort(bson.D{
		{Key: "anio", Value: -1}, {Key: "mes", Value: -1}, {Key: "monto_limite", Value: -1},
	})

	cursor, err := r.coleccion.Find(ctx, consulta, orden)
	if err != nil {
		return nil, traducir(err, "al listar los presupuestos")
	}

	presupuestos := []modelos.Presupuesto{}
	if err := cursor.All(ctx, &presupuestos); err != nil {
		return nil, traducir(err, "al leer los presupuestos")
	}
	return presupuestos, nil
}

// PorID busca un presupuesto del usuario.
func (r *Presupuestos) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Presupuesto, error) {
	var presupuesto modelos.Presupuesto
	if err := r.coleccion.FindOne(ctx, suyoPorID(usuarioID, id)).Decode(&presupuesto); err != nil {
		return nil, traducir(err, "al buscar el presupuesto")
	}
	return &presupuesto, nil
}

// Actualizar reemplaza los campos editables y devuelve el presupuesto cambiado.
//
// Tambien puede chocar con el indice unico: mover un presupuesto al periodo de
// otro que ya existe es un duplicado igual que crearlo.
func (r *Presupuestos) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.Presupuesto) (*modelos.Presupuesto, error) {
	cambios := bson.M{"$set": bson.M{
		"categoria_id": p.CategoriaID,
		"monto_limite": p.MontoLimite,
		"mes":          p.Mes,
		"anio":         p.Anio,
	}}

	opciones := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var actualizado modelos.Presupuesto
	err := r.coleccion.FindOneAndUpdate(ctx, suyoPorID(usuarioID, id), cambios, opciones).Decode(&actualizado)
	if err != nil {
		return nil, traducir(err, "al actualizar el presupuesto")
	}
	return &actualizado, nil
}

// Eliminar borra el presupuesto del usuario.
func (r *Presupuestos) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	resultado, err := r.coleccion.DeleteOne(ctx, suyoPorID(usuarioID, id))
	if err != nil {
		return traducir(err, "al eliminar el presupuesto")
	}
	if resultado.DeletedCount == 0 {
		return ErrNoEncontrado
	}
	return nil
}

// ContarPorCategoria dice cuantos presupuestos usan esa categoria. Lo consulta
// el servicio de categorias antes de dejar borrarla.
func (r *Presupuestos) ContarPorCategoria(ctx context.Context, usuarioID, categoriaID bson.ObjectID) (int64, error) {
	filtro := deUsuario(usuarioID)
	filtro["categoria_id"] = categoriaID

	total, err := r.coleccion.CountDocuments(ctx, filtro)
	if err != nil {
		return 0, traducir(err, "al contar los presupuestos de la categoria")
	}
	return total, nil
}
