package servicios

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// RepositorioCategorias es lo que el servicio necesita de la capa de datos.
type RepositorioCategorias interface {
	Crear(ctx context.Context, categoria *modelos.Categoria) error
	Listar(ctx context.Context, usuarioID bson.ObjectID, tipo string, incluirArchivadas bool) ([]modelos.Categoria, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Categoria, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, categoria modelos.Categoria) (*modelos.Categoria, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// ContadorPorCategoria permite saber si una categoria tiene movimientos.
type ContadorPorCategoria interface {
	ContarPorCategoria(ctx context.Context, usuarioID, categoriaID bson.ObjectID) (int64, error)
}

// Categorias resuelve el CRUD de categorias.
type Categorias struct {
	categorias    RepositorioCategorias
	transacciones ContadorPorCategoria
	presupuestos  ContadorPorCategoria
}

// NuevoCategorias arma el servicio con sus dependencias.
func NuevoCategorias(categorias RepositorioCategorias, transacciones, presupuestos ContadorPorCategoria) *Categorias {
	return &Categorias{categorias: categorias, transacciones: transacciones, presupuestos: presupuestos}
}

// Crear da de alta una categoria del usuario.
func (s *Categorias) Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionCategoria) (*modelos.Categoria, error) {
	categoria := &modelos.Categoria{
		UsuarioID: usuarioID,
		Nombre:    p.Nombre,
		Tipo:      p.Tipo,
		Color:     p.Color,
		Icono:     p.Icono,
		Archivada: p.Archivada,
	}

	if err := s.categorias.Crear(ctx, categoria); err != nil {
		return nil, errores.Interno(err)
	}
	return categoria, nil
}

// Listar devuelve las categorias del usuario, opcionalmente de un solo tipo.
func (s *Categorias) Listar(ctx context.Context, usuarioID bson.ObjectID, tipo string, incluirArchivadas bool) ([]modelos.Categoria, error) {
	categorias, err := s.categorias.Listar(ctx, usuarioID, tipo, incluirArchivadas)
	if err != nil {
		return nil, errores.Interno(err)
	}
	return categorias, nil
}

// PorID devuelve una categoria del usuario.
func (s *Categorias) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Categoria, error) {
	categoria, err := s.categorias.PorID(ctx, usuarioID, id)
	if err != nil {
		return nil, traducirCategoria(err)
	}
	return categoria, nil
}

// Actualizar cambia los datos de una categoria del usuario.
//
// Ojo con cambiar el tipo de una categoria que ya tiene movimientos: dejaria
// gastos colgando de una categoria de ingreso. Por eso se bloquea.
func (s *Categorias) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionCategoria) (*modelos.Categoria, error) {
	actual, err := s.categorias.PorID(ctx, usuarioID, id)
	if err != nil {
		return nil, traducirCategoria(err)
	}

	if actual.Tipo != p.Tipo {
		movimientos, err := s.transacciones.ContarPorCategoria(ctx, usuarioID, id)
		if err != nil {
			return nil, errores.Interno(err)
		}
		if movimientos > 0 {
			return nil, errores.Conflicto(errores.CodigoCategoriaConTransacciones,
				"No se puede cambiar el tipo de una categoria que ya tiene movimientos.")
		}
	}

	categoria, err := s.categorias.Actualizar(ctx, usuarioID, id, modelos.Categoria{
		Nombre:    p.Nombre,
		Tipo:      p.Tipo,
		Color:     p.Color,
		Icono:     p.Icono,
		Archivada: p.Archivada,
	})
	if err != nil {
		return nil, traducirCategoria(err)
	}
	return categoria, nil
}

// Eliminar borra una categoria, pero solo si nadie la esta usando.
//
// Los presupuestos cuentan igual que los movimientos: la consulta de estado
// cruza presupuestos con categorias y descarta las filas cuya categoria ya no
// existe, asi que un presupuesto huerfano no daria error, simplemente
// desapareceria del tablero sin que nadie se entere.
func (s *Categorias) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	movimientos, err := s.transacciones.ContarPorCategoria(ctx, usuarioID, id)
	if err != nil {
		return errores.Interno(err)
	}
	if movimientos > 0 {
		return errores.Conflicto(errores.CodigoCategoriaConTransacciones,
			"Esta categoria tiene movimientos registrados. Archivala en lugar de borrarla.")
	}

	presupuestos, err := s.presupuestos.ContarPorCategoria(ctx, usuarioID, id)
	if err != nil {
		return errores.Interno(err)
	}
	if presupuestos > 0 {
		return errores.Conflicto(errores.CodigoCategoriaConPresupuestos,
			"Esta categoria tiene presupuestos asignados. Borralos primero o archiva la categoria.")
	}

	if err := s.categorias.Eliminar(ctx, usuarioID, id); err != nil {
		return traducirCategoria(err)
	}
	return nil
}

// traducirCategoria convierte los errores del repositorio en errores de dominio.
func traducirCategoria(err error) error {
	if errors.Is(err, repositorios.ErrNoEncontrado) {
		return errores.NoEncontrado(errores.CodigoCategoriaNoEncontrada, "La categoria no existe.")
	}
	return errores.Interno(err)
}
