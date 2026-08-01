// Package repositorios es la unica capa que consulta MongoDB. Los servicios
// hablan con estas estructuras y nunca con el driver directamente, para poder
// probar las reglas de negocio con dobles en vez de con una base real.
package repositorios

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Errores centinela del repositorio.
//
// El repositorio no devuelve errores de dominio: traduce los errores del driver
// a estos dos, y el servicio decide que significan en terminos de negocio. Asi
// la capa de datos no tiene que saber que codigo HTTP le toca a cada caso.
var (
	ErrNoEncontrado = errors.New("documento no encontrado")
	ErrDuplicado    = errors.New("ya existe un documento con esa llave unica")
)

// Usuarios accede a la coleccion usuarios.
type Usuarios struct {
	coleccion *mongo.Collection
}

// NuevoUsuarios construye el repositorio sobre la base indicada.
func NuevoUsuarios(bd *mongo.Database) *Usuarios {
	return &Usuarios{coleccion: bd.Collection("usuarios")}
}

// Crear inserta el usuario y le asigna el _id que genero MongoDB.
//
// No se comprueba antes si el email existe: se deja que falle el indice unico.
// Comprobar y despues insertar deja una ventana en la que dos registros
// simultaneos con el mismo correo pueden pasar los dos.
func (r *Usuarios) Crear(ctx context.Context, usuario *modelos.Usuario) error {
	resultado, err := r.coleccion.InsertOne(ctx, usuario)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicado
		}
		return fmt.Errorf("al insertar el usuario: %w", err)
	}

	if id, ok := resultado.InsertedID.(bson.ObjectID); ok {
		usuario.ID = id
	}
	return nil
}

// PorEmail busca por correo. Devuelve ErrNoEncontrado si no hay ninguno.
func (r *Usuarios) PorEmail(ctx context.Context, email string) (*modelos.Usuario, error) {
	return r.uno(ctx, bson.M{"email": email})
}

// PorID busca por identificador.
func (r *Usuarios) PorID(ctx context.Context, id bson.ObjectID) (*modelos.Usuario, error) {
	return r.uno(ctx, bson.M{"_id": id})
}

// Actualizar cambia el nombre y la moneda, y devuelve el usuario ya actualizado.
func (r *Usuarios) Actualizar(ctx context.Context, id bson.ObjectID, nombre, moneda string) (*modelos.Usuario, error) {
	cambios := bson.M{"$set": bson.M{"nombre": nombre, "moneda": moneda}}

	// ReturnDocument(After) evita tener que hacer un segundo Find.
	opciones := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var usuario modelos.Usuario
	err := r.coleccion.FindOneAndUpdate(ctx, bson.M{"_id": id}, cambios, opciones).Decode(&usuario)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNoEncontrado
		}
		return nil, fmt.Errorf("al actualizar el usuario: %w", err)
	}
	return &usuario, nil
}

// uno ejecuta un FindOne y traduce el "sin documentos" del driver.
func (r *Usuarios) uno(ctx context.Context, filtro bson.M) (*modelos.Usuario, error) {
	var usuario modelos.Usuario
	err := r.coleccion.FindOne(ctx, filtro).Decode(&usuario)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNoEncontrado
		}
		return nil, fmt.Errorf("al buscar el usuario: %w", err)
	}
	return &usuario, nil
}
