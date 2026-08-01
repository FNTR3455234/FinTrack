package servicios

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// Repositorios en memoria para probar los servicios sin MongoDB.
//
// Importante: filtran por usuario_id igual que los de verdad. Si no lo
// hicieran, las pruebas de aislamiento entre usuarios pasarian aunque el
// servicio estuviera mal.

// --- cuentas ----------------------------------------------------------------

type cuentasFalso struct {
	datos        map[bson.ObjectID]*modelos.Cuenta
	errorForzado error
}

func nuevoCuentasFalso() *cuentasFalso {
	return &cuentasFalso{datos: map[bson.ObjectID]*modelos.Cuenta{}}
}

func (r *cuentasFalso) Crear(_ context.Context, cuenta *modelos.Cuenta) error {
	if r.errorForzado != nil {
		return r.errorForzado
	}
	cuenta.ID = bson.NewObjectID()
	copia := *cuenta
	r.datos[cuenta.ID] = &copia
	return nil
}

func (r *cuentasFalso) Listar(_ context.Context, usuarioID bson.ObjectID, incluirArchivadas bool) ([]modelos.Cuenta, error) {
	if r.errorForzado != nil {
		return nil, r.errorForzado
	}
	cuentas := []modelos.Cuenta{}
	for _, c := range r.datos {
		if c.UsuarioID != usuarioID || (!incluirArchivadas && c.Archivada) {
			continue
		}
		cuentas = append(cuentas, *c)
	}
	sort.Slice(cuentas, func(i, j int) bool { return cuentas[i].Nombre < cuentas[j].Nombre })
	return cuentas, nil
}

func (r *cuentasFalso) PorID(_ context.Context, usuarioID, id bson.ObjectID) (*modelos.Cuenta, error) {
	c, existe := r.datos[id]
	if !existe || c.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	copia := *c
	return &copia, nil
}

func (r *cuentasFalso) Actualizar(_ context.Context, usuarioID, id bson.ObjectID, cambios modelos.Cuenta) (*modelos.Cuenta, error) {
	c, existe := r.datos[id]
	if !existe || c.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	c.Nombre, c.Tipo, c.SaldoInicial = cambios.Nombre, cambios.Tipo, cambios.SaldoInicial
	c.Color, c.Archivada = cambios.Color, cambios.Archivada
	copia := *c
	return &copia, nil
}

func (r *cuentasFalso) Eliminar(_ context.Context, usuarioID, id bson.ObjectID) error {
	c, existe := r.datos[id]
	if !existe || c.UsuarioID != usuarioID {
		return repositorios.ErrNoEncontrado
	}
	delete(r.datos, id)
	return nil
}

func (r *cuentasFalso) Existe(_ context.Context, usuarioID, id bson.ObjectID) (bool, error) {
	if r.errorForzado != nil {
		return false, r.errorForzado
	}
	c, existe := r.datos[id]
	return existe && c.UsuarioID == usuarioID, nil
}

// --- categorias -------------------------------------------------------------

type categoriasFalso struct {
	datos map[bson.ObjectID]*modelos.Categoria
}

func nuevoCategoriasFalso() *categoriasFalso {
	return &categoriasFalso{datos: map[bson.ObjectID]*modelos.Categoria{}}
}

func (r *categoriasFalso) Crear(_ context.Context, categoria *modelos.Categoria) error {
	categoria.ID = bson.NewObjectID()
	copia := *categoria
	r.datos[categoria.ID] = &copia
	return nil
}

func (r *categoriasFalso) Listar(_ context.Context, usuarioID bson.ObjectID, tipo string, incluirArchivadas bool) ([]modelos.Categoria, error) {
	categorias := []modelos.Categoria{}
	for _, c := range r.datos {
		if c.UsuarioID != usuarioID || (!incluirArchivadas && c.Archivada) {
			continue
		}
		if tipo != "" && c.Tipo != tipo {
			continue
		}
		categorias = append(categorias, *c)
	}
	sort.Slice(categorias, func(i, j int) bool { return categorias[i].Nombre < categorias[j].Nombre })
	return categorias, nil
}

func (r *categoriasFalso) PorID(_ context.Context, usuarioID, id bson.ObjectID) (*modelos.Categoria, error) {
	c, existe := r.datos[id]
	if !existe || c.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	copia := *c
	return &copia, nil
}

func (r *categoriasFalso) Actualizar(_ context.Context, usuarioID, id bson.ObjectID, cambios modelos.Categoria) (*modelos.Categoria, error) {
	c, existe := r.datos[id]
	if !existe || c.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	c.Nombre, c.Tipo, c.Color, c.Icono, c.Archivada = cambios.Nombre, cambios.Tipo, cambios.Color, cambios.Icono, cambios.Archivada
	copia := *c
	return &copia, nil
}

func (r *categoriasFalso) Eliminar(_ context.Context, usuarioID, id bson.ObjectID) error {
	c, existe := r.datos[id]
	if !existe || c.UsuarioID != usuarioID {
		return repositorios.ErrNoEncontrado
	}
	delete(r.datos, id)
	return nil
}

// --- transacciones ----------------------------------------------------------

type transaccionesFalso struct {
	datos map[bson.ObjectID]*modelos.Transaccion
	// ultimoFiltro guarda lo que recibio Listar, para comprobar que el handler
	// y el servicio pasan los criterios tal cual.
	ultimoFiltro modelos.FiltroTransacciones
}

func nuevoTransaccionesFalso() *transaccionesFalso {
	return &transaccionesFalso{datos: map[bson.ObjectID]*modelos.Transaccion{}}
}

func (r *transaccionesFalso) Crear(_ context.Context, t *modelos.Transaccion) error {
	t.ID = bson.NewObjectID()
	copia := *t
	r.datos[t.ID] = &copia
	return nil
}

func (r *transaccionesFalso) Listar(_ context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroTransacciones) ([]modelos.Transaccion, int64, error) {
	r.ultimoFiltro = filtro

	transacciones := []modelos.Transaccion{}
	for _, t := range r.datos {
		if t.UsuarioID == usuarioID {
			transacciones = append(transacciones, *t)
		}
	}
	return transacciones, int64(len(transacciones)), nil
}

func (r *transaccionesFalso) PorID(_ context.Context, usuarioID, id bson.ObjectID) (*modelos.Transaccion, error) {
	t, existe := r.datos[id]
	if !existe || t.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	copia := *t
	return &copia, nil
}

func (r *transaccionesFalso) Actualizar(_ context.Context, usuarioID, id bson.ObjectID, cambios modelos.Transaccion) (*modelos.Transaccion, error) {
	t, existe := r.datos[id]
	if !existe || t.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	creadoEn := t.CreadoEn
	cambios.ID, cambios.UsuarioID, cambios.CreadoEn = id, usuarioID, creadoEn
	r.datos[id] = &cambios
	copia := cambios
	return &copia, nil
}

func (r *transaccionesFalso) Eliminar(_ context.Context, usuarioID, id bson.ObjectID) error {
	t, existe := r.datos[id]
	if !existe || t.UsuarioID != usuarioID {
		return repositorios.ErrNoEncontrado
	}
	delete(r.datos, id)
	return nil
}

func (r *transaccionesFalso) ContarPorCuenta(_ context.Context, usuarioID, cuentaID bson.ObjectID) (int64, error) {
	var total int64
	for _, t := range r.datos {
		if t.UsuarioID == usuarioID && t.CuentaID == cuentaID {
			total++
		}
	}
	return total, nil
}

func (r *transaccionesFalso) ContarPorCategoria(_ context.Context, usuarioID, categoriaID bson.ObjectID) (int64, error) {
	var total int64
	for _, t := range r.datos {
		if t.UsuarioID == usuarioID && t.CategoriaID == categoriaID {
			total++
		}
	}
	return total, nil
}
