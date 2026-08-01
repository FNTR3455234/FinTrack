package servicios

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// RepositorioCuentas es lo que el servicio necesita de la capa de datos.
type RepositorioCuentas interface {
	Crear(ctx context.Context, cuenta *modelos.Cuenta) error
	Listar(ctx context.Context, usuarioID bson.ObjectID, incluirArchivadas bool) ([]modelos.Cuenta, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Cuenta, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, cuenta modelos.Cuenta) (*modelos.Cuenta, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// ContadorPorCuenta permite saber si una cuenta tiene movimientos.
type ContadorPorCuenta interface {
	ContarPorCuenta(ctx context.Context, usuarioID, cuentaID bson.ObjectID) (int64, error)
}

// Cuentas resuelve el CRUD de cuentas.
type Cuentas struct {
	cuentas       RepositorioCuentas
	transacciones ContadorPorCuenta
}

// NuevoCuentas arma el servicio con sus dependencias.
func NuevoCuentas(cuentas RepositorioCuentas, transacciones ContadorPorCuenta) *Cuentas {
	return &Cuentas{cuentas: cuentas, transacciones: transacciones}
}

// Crear da de alta una cuenta del usuario.
func (s *Cuentas) Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionCuenta) (*modelos.Cuenta, error) {
	cuenta := &modelos.Cuenta{
		// El dueño sale del token, nunca del cuerpo de la peticion.
		UsuarioID:    usuarioID,
		Nombre:       p.Nombre,
		Tipo:         p.Tipo,
		SaldoInicial: *p.SaldoInicial,
		Color:        p.Color,
		Archivada:    p.Archivada,
	}

	if err := s.cuentas.Crear(ctx, cuenta); err != nil {
		return nil, errores.Interno(err)
	}
	return cuenta, nil
}

// Listar devuelve las cuentas del usuario.
func (s *Cuentas) Listar(ctx context.Context, usuarioID bson.ObjectID, incluirArchivadas bool) ([]modelos.Cuenta, error) {
	cuentas, err := s.cuentas.Listar(ctx, usuarioID, incluirArchivadas)
	if err != nil {
		return nil, errores.Interno(err)
	}
	return cuentas, nil
}

// PorID devuelve una cuenta del usuario.
func (s *Cuentas) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Cuenta, error) {
	cuenta, err := s.cuentas.PorID(ctx, usuarioID, id)
	if err != nil {
		return nil, traducirCuenta(err)
	}
	return cuenta, nil
}

// Actualizar cambia los datos de una cuenta del usuario.
func (s *Cuentas) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionCuenta) (*modelos.Cuenta, error) {
	cuenta, err := s.cuentas.Actualizar(ctx, usuarioID, id, modelos.Cuenta{
		Nombre:       p.Nombre,
		Tipo:         p.Tipo,
		SaldoInicial: *p.SaldoInicial,
		Color:        p.Color,
		Archivada:    p.Archivada,
	})
	if err != nil {
		return nil, traducirCuenta(err)
	}
	return cuenta, nil
}

// Eliminar borra una cuenta, pero solo si no tiene movimientos.
//
// No se borra en cascada a proposito: perder las transacciones de una cuenta
// por equivocacion no tiene vuelta atras. Si tiene movimientos, se responde 409
// y el cliente puede archivarla, que es lo que sirve el campo archivada.
func (s *Cuentas) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	movimientos, err := s.transacciones.ContarPorCuenta(ctx, usuarioID, id)
	if err != nil {
		return errores.Interno(err)
	}
	if movimientos > 0 {
		return errores.Conflicto(errores.CodigoCuentaConTransacciones,
			"Esta cuenta tiene movimientos registrados. Archivala en lugar de borrarla.")
	}

	if err := s.cuentas.Eliminar(ctx, usuarioID, id); err != nil {
		return traducirCuenta(err)
	}
	return nil
}

// traducirCuenta convierte los errores del repositorio en errores de dominio.
func traducirCuenta(err error) error {
	if errors.Is(err, repositorios.ErrNoEncontrado) {
		return errores.NoEncontrado(errores.CodigoCuentaNoEncontrada, "La cuenta no existe.")
	}
	return errores.Interno(err)
}
