// Las consultas de reporte de FinTrack. A diferencia del resto del paquete,
// aqui no se leen documentos sueltos: se ejecutan agregaciones que cruzan
// colecciones y devuelven filas calculadas.
//
// Las dos consultas relacionales de la entrega estan en reportes_gastos.go y en
// reportes_presupuestos.go, cada una con su objetivo documentado. Las mismas
// consultas, en su version de mongosh, estan en database/README.md.
package repositorios

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// Reportes ejecuta las agregaciones. Guarda las tres colecciones que cruza
// porque cada agregacion arranca en una distinta.
type Reportes struct {
	transacciones *mongo.Collection
	presupuestos  *mongo.Collection
	cuentas       *mongo.Collection
}

// NuevoReportes construye el repositorio sobre la base indicada.
func NuevoReportes(bd *mongo.Database) *Reportes {
	return &Reportes{
		transacciones: bd.Collection("transacciones"),
		presupuestos:  bd.Collection("presupuestos"),
		cuentas:       bd.Collection("cuentas"),
	}
}

// agregar ejecuta un pipeline y vuelca el resultado en destino, que tiene que
// ser un puntero a slice.
//
// Se centraliza aqui para que ninguna agregacion se olvide de cerrar el cursor
// ni de revisar el error del final del recorrido.
func agregar(ctx context.Context, coleccion *mongo.Collection, etapas []bson.M, destino any, operacion string) error {
	cursor, err := coleccion.Aggregate(ctx, etapas)
	if err != nil {
		return traducir(err, operacion)
	}
	defer func() { _ = cursor.Close(ctx) }()

	if err := cursor.All(ctx, destino); err != nil {
		return traducir(err, operacion)
	}
	return nil
}

// deUsuarioEnPeriodo es el $match con el que arrancan casi todas las
// agregaciones: los movimientos del usuario dentro de un rango de fechas.
//
// El usuario_id va SIEMPRE y es lo primero de la etapa: es lo que impide que un
// reporte sume dinero ajeno, y ademas deja que el indice
// (usuario_id, fecha DESC) recorte el conjunto antes de calcular nada.
func deUsuarioEnPeriodo(usuarioID bson.ObjectID, periodo modelos.Periodo) bson.M {
	inicio, fin := periodo.Rango()
	return bson.M{
		"usuario_id": usuarioID,
		"fecha":      bson.M{"$gte": inicio, "$lt": fin},
	}
}
