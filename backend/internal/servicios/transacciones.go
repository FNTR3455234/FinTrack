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

// RepositorioTransacciones es lo que el servicio necesita de la capa de datos.
type RepositorioTransacciones interface {
	Crear(ctx context.Context, t *modelos.Transaccion) error
	Listar(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroTransacciones) ([]modelos.Transaccion, int64, error)
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Transaccion, error)
	Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, t modelos.Transaccion) (*modelos.Transaccion, error)
	Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error
}

// ComprobadorCuenta dice si una cuenta pertenece al usuario.
type ComprobadorCuenta interface {
	Existe(ctx context.Context, usuarioID, id bson.ObjectID) (bool, error)
}

// BuscadorCategoria devuelve una categoria del usuario. Hace falta el documento
// completo, no solo saber si existe, porque se compara su tipo.
type BuscadorCategoria interface {
	PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Categoria, error)
}

// Transacciones resuelve el CRUD de transacciones.
type Transacciones struct {
	transacciones RepositorioTransacciones
	cuentas       ComprobadorCuenta
	categorias    BuscadorCategoria
	presupuestos  EvaluadorPresupuesto
}

// NuevoTransacciones arma el servicio con sus dependencias.
func NuevoTransacciones(t RepositorioTransacciones, c ComprobadorCuenta, cat BuscadorCategoria, p EvaluadorPresupuesto) *Transacciones {
	return &Transacciones{transacciones: t, cuentas: c, categorias: cat, presupuestos: p}
}

// Crear registra un movimiento y, si es un gasto, revisa el presupuesto de su
// categoria para avisar cuando queda cerca del limite o por encima.
//
// La alerta se calcula DESPUES de guardar, a proposito: asi el total que
// reporta ya incluye el movimiento que el usuario acaba de capturar, que es lo
// que espera ver.
func (s *Transacciones) Crear(ctx context.Context, usuarioID bson.ObjectID, p modelos.PeticionTransaccion) (*modelos.TransaccionCreada, error) {
	cuentaID, categoriaID, err := s.validarReferencias(ctx, usuarioID, p)
	if err != nil {
		return nil, err
	}

	momento := ahora()
	transaccion := &modelos.Transaccion{
		UsuarioID:     usuarioID,
		CuentaID:      cuentaID,
		CategoriaID:   categoriaID,
		Tipo:          p.Tipo,
		Monto:         redondear(p.Monto),
		Descripcion:   strings.TrimSpace(p.Descripcion),
		Fecha:         diaCalendario(p.Fecha),
		Notas:         notasONulo(p.Notas),
		CreadoEn:      momento,
		ActualizadoEn: momento,
	}

	if err := s.transacciones.Crear(ctx, transaccion); err != nil {
		return nil, errores.Interno(err)
	}

	return &modelos.TransaccionCreada{
		Transaccion: *transaccion,
		Alerta:      s.alertaDe(ctx, usuarioID, transaccion),
	}, nil
}

// Listar devuelve una pagina de transacciones y el total del filtro.
func (s *Transacciones) Listar(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroTransacciones) ([]modelos.Transaccion, int64, error) {
	transacciones, total, err := s.transacciones.Listar(ctx, usuarioID, filtro)
	if err != nil {
		return nil, 0, errores.Interno(err)
	}
	return transacciones, total, nil
}

// PorID devuelve una transaccion del usuario.
func (s *Transacciones) PorID(ctx context.Context, usuarioID, id bson.ObjectID) (*modelos.Transaccion, error) {
	transaccion, err := s.transacciones.PorID(ctx, usuarioID, id)
	if err != nil {
		return nil, traducirTransaccion(err)
	}
	return transaccion, nil
}

// Actualizar cambia un movimiento existente.
func (s *Transacciones) Actualizar(ctx context.Context, usuarioID, id bson.ObjectID, p modelos.PeticionTransaccion) (*modelos.Transaccion, error) {
	cuentaID, categoriaID, err := s.validarReferencias(ctx, usuarioID, p)
	if err != nil {
		return nil, err
	}

	actualizada, err := s.transacciones.Actualizar(ctx, usuarioID, id, modelos.Transaccion{
		CuentaID:      cuentaID,
		CategoriaID:   categoriaID,
		Tipo:          p.Tipo,
		Monto:         redondear(p.Monto),
		Descripcion:   strings.TrimSpace(p.Descripcion),
		Fecha:         diaCalendario(p.Fecha),
		Notas:         notasONulo(p.Notas),
		ActualizadoEn: ahora(),
	})
	if err != nil {
		return nil, traducirTransaccion(err)
	}
	return actualizada, nil
}

// Eliminar borra un movimiento del usuario. Aqui si se borra de verdad: una
// transaccion no es referencia de nada mas.
func (s *Transacciones) Eliminar(ctx context.Context, usuarioID, id bson.ObjectID) error {
	if err := s.transacciones.Eliminar(ctx, usuarioID, id); err != nil {
		return traducirTransaccion(err)
	}
	return nil
}

// traducirTransaccion convierte los errores del repositorio en errores de dominio.
func traducirTransaccion(err error) error {
	if errors.Is(err, repositorios.ErrNoEncontrado) {
		return errores.NoEncontrado(errores.CodigoTransaccionNoEncontrada, "La transaccion no existe.")
	}
	return errores.Interno(err)
}
