package servicios

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// Aportar registra dinero apartado para una meta.
//
// Se comprueba antes que la meta exista y sea del usuario. Sin esa comprobacion
// se podrian crear aportaciones colgando de un meta_id inventado: no se verian
// en ningun sitio, pero ahi quedarian.
//
// No se rechaza pasarse del objetivo. Juntar mas de lo que se planeaba no es un
// error, y bloquearlo obligaria a editar la meta para poder registrar dinero que
// ya se aparto de verdad.
func (s *Metas) Aportar(ctx context.Context, usuarioID, metaID bson.ObjectID, p modelos.PeticionAportacion) (*modelos.Aportacion, error) {
	if _, err := s.metas.PorID(ctx, usuarioID, metaID); err != nil {
		return nil, traducirMeta(err)
	}

	aportacion := &modelos.Aportacion{
		UsuarioID: usuarioID,
		MetaID:    metaID,
		Monto:     redondear(p.Monto),
		// Mismo anclaje que la fecha de una transaccion (decision 019).
		Fecha:    diaCalendario(p.Fecha),
		Nota:     notasONulo(p.Nota),
		CreadoEn: ahora(),
	}

	if err := s.aportaciones.Crear(ctx, aportacion); err != nil {
		return nil, errores.Interno(err)
	}
	return aportacion, nil
}

// QuitarAportacion borra una aportacion concreta de una meta.
//
// El repositorio filtra por meta ademas de por aportacion y usuario, asi que
// pasar el id de una aportacion de otra meta responde 404 en vez de borrarla.
func (s *Metas) QuitarAportacion(ctx context.Context, usuarioID, metaID, aportacionID bson.ObjectID) error {
	err := s.aportaciones.Eliminar(ctx, usuarioID, metaID, aportacionID)
	if err != nil {
		if errors.Is(err, repositorios.ErrNoEncontrado) {
			return errores.NoEncontrado(errores.CodigoAportacionNoEncontrada, "La aportacion no existe.")
		}
		return errores.Interno(err)
	}
	return nil
}
