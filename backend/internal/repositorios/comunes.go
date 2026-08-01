package repositorios

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// deUsuario arma el filtro base de cualquier consulta.
//
// TODA consulta de este paquete pasa por aqui o incluye usuario_id a mano. Es
// lo que impide que un usuario alcance documentos de otro: si el _id existe
// pero es de alguien mas, el filtro no encuentra nada y la operacion responde
// "no encontrado", igual que si no existiera.
func deUsuario(usuarioID bson.ObjectID) bson.M {
	return bson.M{"usuario_id": usuarioID}
}

// suyoPorID filtra por identificador y dueño a la vez.
func suyoPorID(usuarioID, id bson.ObjectID) bson.M {
	return bson.M{"_id": id, "usuario_id": usuarioID}
}

// traducir convierte los errores del driver a los centinelas del paquete.
// El resto se envuelve con contexto para que quede util en la bitacora.
func traducir(err error, operacion string) error {
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return ErrNoEncontrado
	case mongo.IsDuplicateKeyError(err):
		return ErrDuplicado
	default:
		return fmt.Errorf("%s: %w", operacion, err)
	}
}
