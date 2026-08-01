package db

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CrearIndices deja los indices de FinTrack en su lugar al arrancar el servidor.
//
// Son exactamente los mismos que crea database/01_crear_colecciones.js. Se
// repiten aqui a proposito: el script sirve para la base local y para la
// entrega, y este codigo garantiza que un despliegue nuevo (o una base que
// alguien creo a mano) tenga los indices sin depender de que se haya corrido
// el script.
//
// createIndex es idempotente mientras el nombre y las llaves coincidan, asi que
// esta funcion se puede llamar en cada arranque sin costo.
func CrearIndices(ctx context.Context, bd *mongo.Database) error {
	indices := map[string][]mongo.IndexModel{
		// El email es la credencial de acceso: no se puede repetir.
		"usuarios": {{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_usuarios_email_unico"),
		}},

		// Todo listado filtra por dueño.
		"cuentas": {{
			Keys:    bson.D{{Key: "usuario_id", Value: 1}},
			Options: options.Index().SetName("idx_cuentas_usuario"),
		}},
		"categorias": {{
			Keys:    bson.D{{Key: "usuario_id", Value: 1}},
			Options: options.Index().SetName("idx_categorias_usuario"),
		}},

		"transacciones": {
			// Listado principal: filtra por usuario y ordena por fecha
			// descendente con el mismo indice.
			{
				Keys:    bson.D{{Key: "usuario_id", Value: 1}, {Key: "fecha", Value: -1}},
				Options: options.Index().SetName("idx_transacciones_usuario_fecha"),
			},
			// Lo usan el reporte de gastos por categoria y el $lookup de
			// presupuestos.
			{
				Keys:    bson.D{{Key: "usuario_id", Value: 1}, {Key: "categoria_id", Value: 1}},
				Options: options.Index().SetName("idx_transacciones_usuario_categoria"),
			},
		},

		// Un solo presupuesto por categoria y periodo.
		"presupuestos": {{
			Keys: bson.D{
				{Key: "usuario_id", Value: 1},
				{Key: "categoria_id", Value: 1},
				{Key: "mes", Value: 1},
				{Key: "anio", Value: 1},
			},
			Options: options.Index().SetUnique(true).SetName("idx_presupuestos_unico_periodo"),
		}},
	}

	total := 0
	for coleccion, modelos := range indices {
		nombres, err := bd.Collection(coleccion).Indexes().CreateMany(ctx, modelos)
		if err != nil {
			return fmt.Errorf("no se pudieron crear los indices de %q: %w", coleccion, err)
		}
		total += len(nombres)
	}

	slog.Info("indices verificados", "cantidad", total)
	return nil
}
