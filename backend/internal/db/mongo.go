// Package db abre y cierra la conexion con MongoDB y crea los indices.
// Es la unica parte del backend que sabe como se llega a la base; los
// repositorios reciben ya la conexion hecha.
package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Conexion agrupa el cliente (la conexion al servidor) y la base concreta que
// usa la aplicacion. El cliente hace falta para cerrar y para hacer ping; la
// base es lo que consumen los repositorios.
type Conexion struct {
	Cliente *mongo.Client
	BD      *mongo.Database
}

// Conectar abre la conexion y comprueba que el servidor responde.
//
// mongo.Connect no contacta al servidor: solo prepara el cliente. Por eso se
// hace un ping enseguida, para que un URI mal escrito o una base apagada se
// noten al arrancar y no en la primera peticion de un usuario.
func Conectar(ctx context.Context, uri, nombreBD string) (*Conexion, error) {
	opciones := options.Client().
		ApplyURI(uri).
		// Si el servidor no aparece en 10 segundos, se falla en vez de esperar
		// el tiempo por defecto (30 s), que hace parecer que el arranque colgo.
		SetServerSelectionTimeout(10 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetAppName("fintrack-api")

	cliente, err := mongo.Connect(opciones)
	if err != nil {
		return nil, fmt.Errorf("no se pudo preparar el cliente de MongoDB: %w", err)
	}

	conexion := &Conexion{Cliente: cliente, BD: cliente.Database(nombreBD)}

	if err := conexion.Ping(ctx); err != nil {
		// Si el ping falla hay que soltar el cliente igual, para no dejar
		// goroutines del driver dando vueltas.
		_ = cliente.Disconnect(context.Background())
		return nil, fmt.Errorf("no hay respuesta de MongoDB en %s: %w", nombreBD, err)
	}

	return conexion, nil
}

// Ping comprueba que el servidor primario responde. Lo usa /health.
func (c *Conexion) Ping(ctx context.Context) error {
	return c.Cliente.Ping(ctx, readpref.Primary())
}

// Cerrar suelta la conexion. Se llama en el apagado ordenado.
func (c *Conexion) Cerrar(ctx context.Context) error {
	return c.Cliente.Disconnect(ctx)
}
