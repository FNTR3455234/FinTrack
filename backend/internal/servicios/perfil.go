package servicios

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// Perfil devuelve los datos del usuario del token.
//
// El usuarioID llega como argumento desde el handler, que a su vez lo saco del
// contexto donde lo dejo el middleware de autenticacion. Nunca del cuerpo.
func (a *Auth) Perfil(ctx context.Context, usuarioID bson.ObjectID) (*modelos.Usuario, error) {
	usuario, err := a.usuarios.PorID(ctx, usuarioID)
	if err != nil {
		if errors.Is(err, repositorios.ErrNoEncontrado) {
			return nil, errores.NoEncontrado(errores.CodigoUsuarioNoEncontrado,
				"El usuario no existe.")
		}
		return nil, errores.Interno(err)
	}
	return usuario, nil
}

// ActualizarPerfil cambia el nombre y la moneda.
//
// El correo no se toca: es la credencial de acceso y la llave unica, y
// cambiarlo obligaria a revisar que el nuevo no este tomado y a decidir que
// pasa con las sesiones abiertas. Queda fuera del alcance a proposito.
func (a *Auth) ActualizarPerfil(ctx context.Context, usuarioID bson.ObjectID, peticion modelos.PeticionActualizarPerfil) (*modelos.Usuario, error) {
	usuario, err := a.usuarios.Actualizar(ctx, usuarioID,
		strings.TrimSpace(peticion.Nombre), strings.ToUpper(peticion.Moneda))
	if err != nil {
		if errors.Is(err, repositorios.ErrNoEncontrado) {
			return nil, errores.NoEncontrado(errores.CodigoUsuarioNoEncontrado, "El usuario no existe.")
		}
		return nil, errores.Interno(err)
	}
	return usuario, nil
}
