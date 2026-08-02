package servicios

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// RepositorioMetas es lo que el servicio necesita de la coleccion metas.
type RepositorioMetas interface {
	Crear(ctx context.Context, m *modelos.Meta) error
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Meta, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, m modelos.Meta) (*modelos.Meta, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// RepositorioAportaciones es lo que el servicio necesita de la coleccion
// aportaciones.
type RepositorioAportaciones interface {
	Crear(ctx context.Context, a *modelos.Aportacion) error
	DeMeta(ctx context.Context, usuarioID, metaID bson.ObjectID) ([]modelos.Aportacion, error)
	Eliminar(ctx context.Context, usuarioID, metaID, id bson.ObjectID) error
	EliminarDeMeta(ctx context.Context, usuarioID, metaID bson.ObjectID) (int64, error)
}

// CalculadorProgreso es la agregacion que cruza metas con aportaciones.
type CalculadorProgreso interface {
	ProgresoMetas(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroMetas, soloMeta *bson.ObjectID) ([]modelos.ProgresoMeta, error)
}

// Metas resuelve el CRUD de metas de ahorro y sus aportaciones.
type Metas struct {
	metas        RepositorioMetas
	aportaciones RepositorioAportaciones
	progreso     CalculadorProgreso
	reloj        Reloj
}

// NuevoMetas arma el servicio con sus dependencias.
//
// El reloj se inyecta para poder fijarlo en las pruebas. Si viene nil se usa el
// de verdad, asi que quien lo cablea en main no tiene que preocuparse.
func NuevoMetas(m RepositorioMetas, a RepositorioAportaciones, p CalculadorProgreso, reloj Reloj) *Metas {
	if reloj == nil {
		reloj = func() time.Time { return time.Now().UTC() }
	}
	return &Metas{metas: m, aportaciones: a, progreso: p, reloj: reloj}
}

// Crear da de alta una meta de ahorro.
func (s *Metas) Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionMeta) (*modelos.Meta, error) {
	momento := ahora()
	meta := &modelos.Meta{
		// El dueño sale del token, nunca del cuerpo de la peticion.
		UsuarioID:     usuarioID,
		Nombre:        p.Nombre,
		MontoObjetivo: redondear(p.MontoObjetivo),
		// La fecha limite es un dia, no un instante: mismo anclaje que la fecha
		// de una transaccion (ver decision 019).
		FechaLimite:   diaCalendario(p.FechaLimite),
		Color:         p.Color,
		Notas:         notasONulo(p.Notas),
		Archivada:     p.Archivada,
		CreadoEn:      momento,
		ActualizadoEn: momento,
	}

	if err := s.metas.Crear(ctx, meta); err != nil {
		return nil, traducirMeta(err)
	}
	return meta, nil
}

// Listar devuelve las metas del usuario con su progreso y el resumen del
// conjunto.
func (s *Metas) Listar(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroMetas) ([]modelos.ProgresoMeta, modelos.ResumenMetas, error) {
	progreso, err := s.progreso.ProgresoMetas(ctx, usuarioID, filtro, nil)
	if err != nil {
		return nil, modelos.ResumenMetas{}, errores.Interno(err)
	}

	completadas := conCalendario(progreso, s.reloj())
	return completadas, resumirMetas(completadas), nil
}

// PorID devuelve una meta con su progreso y el detalle de sus aportaciones.
//
// El detalle es lo que explica de donde sale la cifra: sin el, el usuario ve un
// total y tiene que creerselo.
func (s *Metas) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.MetaConAportaciones, error) {
	progreso, err := s.progreso.ProgresoMetas(ctx, usuarioID, modelos.FiltroMetas{}, &id)
	if err != nil {
		return nil, errores.Interno(err)
	}
	if len(progreso) == 0 {
		// La agregacion filtra por usuario, asi que una meta ajena simplemente
		// no aparece: para el intruso no existe.
		return nil, errores.NoEncontrado(errores.CodigoMetaNoEncontrada, "La meta no existe.")
	}

	detalle, err := s.aportaciones.DeMeta(ctx, usuarioID, id)
	if err != nil {
		return nil, errores.Interno(err)
	}

	completadas := conCalendario(progreso, s.reloj())
	return &modelos.MetaConAportaciones{ProgresoMeta: completadas[0], Detalle: detalle}, nil
}

// Actualizar cambia los datos de una meta. Las aportaciones no se tocan: subir
// el objetivo no borra lo ya ahorrado, solo cambia el porcentaje.
func (s *Metas) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionMeta) (*modelos.Meta, error) {
	actualizada, err := s.metas.Actualizar(ctx, usuarioID, id, modelos.Meta{
		Nombre:        p.Nombre,
		MontoObjetivo: redondear(p.MontoObjetivo),
		FechaLimite:   diaCalendario(p.FechaLimite),
		Color:         p.Color,
		Notas:         notasONulo(p.Notas),
		Archivada:     p.Archivada,
		ActualizadoEn: ahora(),
	})
	if err != nil {
		return nil, traducirMeta(err)
	}
	return actualizada, nil
}

// Eliminar borra la meta y sus aportaciones.
//
// Aqui SI hay borrado en cascada, al reves que con cuentas y categorias. La
// diferencia es lo que significa cada relacion: una transaccion existe por si
// misma y solo se apoya en su categoria, asi que borrar la categoria la dejaria
// huerfana. Una aportacion, en cambio, no significa nada sin su meta: "3 000 el
// 15 de marzo" no es un dato, es parte de la meta.
//
// El orden importa. Se borran primero las aportaciones y despues la meta porque
// MongoDB esta en modo standalone y no hay transacciones de varios documentos
// (decision 003): si el segundo paso falla, queda una meta con cero
// aportaciones —visible, y se puede volver a borrar— en vez de aportaciones
// huerfanas que ya nadie ve ni puede limpiar.
func (s *Metas) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	// Se comprueba antes que la meta sea del usuario: si no existe, no hay que
	// borrar ninguna aportacion.
	if _, err := s.metas.PorID(ctx, usuarioID, id); err != nil {
		return traducirMeta(err)
	}

	if _, err := s.aportaciones.EliminarDeMeta(ctx, usuarioID, id); err != nil {
		return errores.Interno(err)
	}

	if err := s.metas.Eliminar(ctx, usuarioID, id); err != nil {
		return traducirMeta(err)
	}
	return nil
}

// traducirMeta convierte los errores del repositorio en errores de dominio.
func traducirMeta(err error) error {
	if errors.Is(err, repositorios.ErrNoEncontrado) {
		return errores.NoEncontrado(errores.CodigoMetaNoEncontrada, "La meta no existe.")
	}
	return errores.Interno(err)
}
