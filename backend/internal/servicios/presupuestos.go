package servicios

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// RepositorioPresupuestos es lo que el servicio necesita de la capa de datos.
type RepositorioPresupuestos interface {
	Crear(ctx context.Context, p *modelos.Presupuesto) error
	Listar(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroPresupuestos) ([]modelos.Presupuesto, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Presupuesto, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.Presupuesto) (*modelos.Presupuesto, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// Presupuestos resuelve el CRUD de presupuestos.
type Presupuestos struct {
	presupuestos RepositorioPresupuestos
	categorias   BuscadorCategoria
}

// NuevoPresupuestos arma el servicio con sus dependencias.
func NuevoPresupuestos(p RepositorioPresupuestos, c BuscadorCategoria) *Presupuestos {
	return &Presupuestos{presupuestos: p, categorias: c}
}

// Crear da de alta el limite de una categoria para un mes.
func (s *Presupuestos) Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionPresupuesto) (*modelos.Presupuesto, error) {
	categoriaID, err := s.validarCategoria(ctx, usuarioID, p.CategoriaID)
	if err != nil {
		return nil, err
	}

	presupuesto := &modelos.Presupuesto{
		// El dueño sale del token, nunca del cuerpo de la peticion.
		UsuarioID:   usuarioID,
		CategoriaID: categoriaID,
		MontoLimite: redondear(p.MontoLimite),
		Mes:         p.Mes,
		Anio:        p.Anio,
	}

	if err := s.presupuestos.Crear(ctx, presupuesto); err != nil {
		return nil, traducirPresupuesto(err)
	}
	return presupuesto, nil
}

// Listar devuelve los presupuestos del usuario, opcionalmente los de un periodo.
func (s *Presupuestos) Listar(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroPresupuestos) ([]modelos.Presupuesto, error) {
	presupuestos, err := s.presupuestos.Listar(ctx, usuarioID, filtro)
	if err != nil {
		return nil, errores.Interno(err)
	}
	return presupuestos, nil
}

// PorID devuelve un presupuesto del usuario.
func (s *Presupuestos) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Presupuesto, error) {
	presupuesto, err := s.presupuestos.PorID(ctx, usuarioID, id)
	if err != nil {
		return nil, traducirPresupuesto(err)
	}
	return presupuesto, nil
}

// Actualizar cambia el limite, la categoria o el periodo de un presupuesto.
func (s *Presupuestos) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionPresupuesto) (*modelos.Presupuesto, error) {
	categoriaID, err := s.validarCategoria(ctx, usuarioID, p.CategoriaID)
	if err != nil {
		return nil, err
	}

	actualizado, err := s.presupuestos.Actualizar(ctx, usuarioID, id, modelos.Presupuesto{
		CategoriaID: categoriaID,
		MontoLimite: redondear(p.MontoLimite),
		Mes:         p.Mes,
		Anio:        p.Anio,
	})
	if err != nil {
		return nil, traducirPresupuesto(err)
	}
	return actualizado, nil
}

// Eliminar borra un presupuesto del usuario. Se borra sin mas: un presupuesto
// no es referencia de nada, solo se compara contra las transacciones.
func (s *Presupuestos) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	if err := s.presupuestos.Eliminar(ctx, usuarioID, id); err != nil {
		return traducirPresupuesto(err)
	}
	return nil
}

// validarCategoria comprueba que la categoria exista, sea del usuario y sea de
// gasto.
//
// El categoria_id llega en el cuerpo, o sea que lo elige el cliente: como la
// consulta filtra por usuario_id, una categoria ajena simplemente "no existe".
//
// Solo se presupuestan gastos: ponerle un techo a un ingreso no significa nada,
// y ademas la consulta de estado suma unicamente transacciones de tipo gasto,
// asi que un presupuesto sobre una categoria de ingreso se veria siempre en 0.
func (s *Presupuestos) validarCategoria(ctx context.Context, usuarioID bson.ObjectID, hexadecimal string) (bson.ObjectID, error) {
	categoriaID, err := bson.ObjectIDFromHex(hexadecimal)
	if err != nil {
		return bson.NilObjectID, errores.SolicitudInvalida(
			errores.CodigoIDInvalido, "El identificador de la categoria no es valido.")
	}

	categoria, err := s.categorias.PorID(ctx, usuarioID, categoriaID)
	if err != nil {
		if errors.Is(err, repositorios.ErrNoEncontrado) {
			return bson.NilObjectID, errores.NoEncontrado(
				errores.CodigoCategoriaNoEncontrada, "La categoria no existe.")
		}
		return bson.NilObjectID, errores.Interno(err)
	}

	if categoria.Tipo != modelos.TipoGasto {
		return bson.NilObjectID, errores.SolicitudInvalida(errores.CodigoTipoNoCoincide,
			"Solo se puede presupuestar una categoria de gasto.").
			ConDetalles("la categoria \"" + categoria.Nombre + "\" es de tipo \"" + categoria.Tipo + "\"")
	}

	return categoriaID, nil
}

// traducirPresupuesto convierte los errores del repositorio en errores de dominio.
func traducirPresupuesto(err error) error {
	switch {
	case errors.Is(err, repositorios.ErrNoEncontrado):
		return errores.NoEncontrado(errores.CodigoPresupuestoNoEncontrado, "El presupuesto no existe.")
	case errors.Is(err, repositorios.ErrDuplicado):
		// Lo decide el indice unico (usuario_id, categoria_id, mes, anio).
		return errores.Conflicto(errores.CodigoPresupuestoDuplicado,
			"Ya existe un presupuesto para esa categoria en ese mes.")
	default:
		return errores.Interno(err)
	}
}
