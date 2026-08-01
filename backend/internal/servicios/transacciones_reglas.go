package servicios

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// validarReferencias comprueba que la cuenta y la categoria existan, sean del
// usuario, y que el tipo del movimiento cuadre con el de su categoria.
//
// Es la comprobacion mas importante del CRUD: cuenta_id y categoria_id llegan
// en el cuerpo, o sea que los elige el cliente. Si no se verificara que son del
// usuario del token, cualquiera podria colgar un movimiento de la cuenta de
// otro con solo mandar su identificador. Como las dos consultas filtran por
// usuario_id, un id ajeno simplemente "no existe".
func (s *Transacciones) validarReferencias(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionTransaccion) (bson.ObjectID, bson.ObjectID, error) {
	cuentaID, err := bson.ObjectIDFromHex(p.CuentaID)
	if err != nil {
		return bson.NilObjectID, bson.NilObjectID, errores.SolicitudInvalida(
			errores.CodigoIDInvalido, "El identificador de la cuenta no es valido.")
	}

	categoriaID, err := bson.ObjectIDFromHex(p.CategoriaID)
	if err != nil {
		return bson.NilObjectID, bson.NilObjectID, errores.SolicitudInvalida(
			errores.CodigoIDInvalido, "El identificador de la categoria no es valido.")
	}

	existe, err := s.cuentas.Existe(ctx, usuarioID, cuentaID)
	if err != nil {
		return bson.NilObjectID, bson.NilObjectID, errores.Interno(err)
	}
	if !existe {
		return bson.NilObjectID, bson.NilObjectID, errores.NoEncontrado(
			errores.CodigoCuentaNoEncontrada, "La cuenta no existe.")
	}

	categoria, err := s.categorias.PorID(ctx, usuarioID, categoriaID)
	if err != nil {
		if errors.Is(err, repositorios.ErrNoEncontrado) {
			return bson.NilObjectID, bson.NilObjectID, errores.NoEncontrado(
				errores.CodigoCategoriaNoEncontrada, "La categoria no existe.")
		}
		return bson.NilObjectID, bson.NilObjectID, errores.Interno(err)
	}

	// El tipo esta duplicado (en la transaccion y en su categoria) para que la
	// consulta de gastos por categoria pueda filtrar sin resolver la categoria
	// de cada documento. El precio de esa duplicacion es mantenerla coherente,
	// y esta es la unica puerta por donde entra.
	if categoria.Tipo != p.Tipo {
		return bson.NilObjectID, bson.NilObjectID, errores.SolicitudInvalida(
			errores.CodigoTipoNoCoincide,
			"El tipo del movimiento no coincide con el de su categoria.",
		).ConDetalles(fmt.Sprintf("la categoria %q es de tipo %q y el movimiento se envio como %q",
			categoria.Nombre, categoria.Tipo, p.Tipo))
	}

	return cuentaID, categoriaID, nil
}

// redondear deja el monto en centavos exactos.
//
// El dinero se guarda como float64 (ver docs/decisiones.md, decision 001).
// Redondear al guardar evita que un 10.999999999 escrito por un cliente termine
// arrastrandose por todas las sumas de los reportes.
func redondear(monto float64) float64 {
	return math.Round(monto*100) / 100
}

// notasONulo devuelve nil cuando las notas vienen vacias, para guardar null en
// MongoDB en vez de una cadena vacia. Asi "sin notas" es un solo valor.
func notasONulo(notas string) *string {
	limpias := strings.TrimSpace(notas)
	if limpias == "" {
		return nil
	}
	return &limpias
}
