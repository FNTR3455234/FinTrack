package servicios

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// --- presupuestos -----------------------------------------------------------

type presupuestosFalso struct {
	datos map[bson.ObjectID]*modelos.Presupuesto
}

func nuevoPresupuestosFalso() *presupuestosFalso {
	return &presupuestosFalso{datos: map[bson.ObjectID]*modelos.Presupuesto{}}
}

// Crear imita tambien el indice unico (usuario_id, categoria_id, mes, anio):
// sin eso, la prueba del 409 pasaria por el camino equivocado.
func (r *presupuestosFalso) Crear(_ context.Context, p *modelos.Presupuesto) error {
	if r.duplicado(p.UsuarioID, p.CategoriaID, p.Periodo(), bson.NilObjectID) {
		return repositorios.ErrDuplicado
	}
	p.ID = bson.NewObjectID()
	copia := *p
	r.datos[p.ID] = &copia
	return nil
}

func (r *presupuestosFalso) duplicado(usuarioID, categoriaID bson.ObjectID, periodo modelos.Periodo, salvo bson.ObjectID) bool {
	for id, p := range r.datos {
		if id == salvo {
			continue
		}
		if p.UsuarioID == usuarioID && p.CategoriaID == categoriaID && p.Periodo() == periodo {
			return true
		}
	}
	return false
}

func (r *presupuestosFalso) Listar(_ context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroPresupuestos) ([]modelos.Presupuesto, error) {
	presupuestos := []modelos.Presupuesto{}
	for _, p := range r.datos {
		if p.UsuarioID != usuarioID {
			continue
		}
		if filtro.Periodo != nil && p.Periodo() != *filtro.Periodo {
			continue
		}
		presupuestos = append(presupuestos, *p)
	}
	sort.Slice(presupuestos, func(i, j int) bool {
		return presupuestos[i].MontoLimite > presupuestos[j].MontoLimite
	})
	return presupuestos, nil
}

func (r *presupuestosFalso) PorID(_ context.Context, usuarioID, id bson.ObjectID) (*modelos.Presupuesto, error) {
	p, existe := r.datos[id]
	if !existe || p.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	copia := *p
	return &copia, nil
}

func (r *presupuestosFalso) Actualizar(_ context.Context, usuarioID, id bson.ObjectID, cambios modelos.Presupuesto) (*modelos.Presupuesto, error) {
	p, existe := r.datos[id]
	if !existe || p.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	if r.duplicado(usuarioID, cambios.CategoriaID, cambios.Periodo(), id) {
		return nil, repositorios.ErrDuplicado
	}
	p.CategoriaID, p.MontoLimite, p.Mes, p.Anio = cambios.CategoriaID, cambios.MontoLimite, cambios.Mes, cambios.Anio
	copia := *p
	return &copia, nil
}

func (r *presupuestosFalso) Eliminar(_ context.Context, usuarioID, id bson.ObjectID) error {
	p, existe := r.datos[id]
	if !existe || p.UsuarioID != usuarioID {
		return repositorios.ErrNoEncontrado
	}
	delete(r.datos, id)
	return nil
}

func (r *presupuestosFalso) ContarPorCategoria(_ context.Context, usuarioID, categoriaID bson.ObjectID) (int64, error) {
	var total int64
	for _, p := range r.datos {
		if p.UsuarioID == usuarioID && p.CategoriaID == categoriaID {
			total++
		}
	}
	return total, nil
}

// --- evaluador de presupuestos ----------------------------------------------

// evaluadorFalso hace en memoria lo que en produccion hace la consulta
// relacional 2: cruza el presupuesto de una categoria con lo gastado en ella
// durante el mes.
//
// Los umbrales salen de las mismas constantes que usa la agregacion, para que
// el doble no pueda quedar semaforo con un criterio distinto.
type evaluadorFalso struct {
	presupuestos  *presupuestosFalso
	transacciones *transaccionesFalso
	categorias    *categoriasFalso
	errorForzado  error
}

func nuevoEvaluadorFalso(p *presupuestosFalso, t *transaccionesFalso, c *categoriasFalso) *evaluadorFalso {
	return &evaluadorFalso{presupuestos: p, transacciones: t, categorias: c}
}

func (e *evaluadorFalso) EstadoDeCategoria(_ context.Context, usuarioID, categoriaID bson.ObjectID, periodo modelos.Periodo) (*modelos.EstadoPresupuesto, error) {
	if e.errorForzado != nil {
		return nil, e.errorForzado
	}

	presupuesto := e.presupuestoDe(usuarioID, categoriaID, periodo)
	if presupuesto == nil {
		return nil, nil
	}

	gastado := redondear(e.gastadoEn(usuarioID, categoriaID, periodo))
	porcentaje := redondear(gastado / presupuesto.MontoLimite * 100)

	var nombre string
	if categoria, existe := e.categorias.datos[categoriaID]; existe {
		nombre = categoria.Nombre
	}

	return &modelos.EstadoPresupuesto{
		PresupuestoID:   presupuesto.ID,
		CategoriaID:     categoriaID,
		Nombre:          nombre,
		MontoLimite:     presupuesto.MontoLimite,
		Gastado:         gastado,
		Disponible:      redondear(presupuesto.MontoLimite - gastado),
		PorcentajeUsado: porcentaje,
		Estado:          estadoSegun(porcentaje),
	}, nil
}

func (e *evaluadorFalso) presupuestoDe(usuarioID, categoriaID bson.ObjectID, periodo modelos.Periodo) *modelos.Presupuesto {
	for _, p := range e.presupuestos.datos {
		if p.UsuarioID == usuarioID && p.CategoriaID == categoriaID && p.Periodo() == periodo {
			return p
		}
	}
	return nil
}

func (e *evaluadorFalso) gastadoEn(usuarioID, categoriaID bson.ObjectID, periodo modelos.Periodo) float64 {
	inicio, fin := periodo.Rango()

	var total float64
	for _, t := range e.transacciones.datos {
		mismoDueño := t.UsuarioID == usuarioID && t.CategoriaID == categoriaID
		enElMes := !t.Fecha.Before(inicio) && t.Fecha.Before(fin)
		if mismoDueño && t.Tipo == modelos.TipoGasto && enElMes {
			total += t.Monto
		}
	}
	return total
}

func estadoSegun(porcentaje float64) string {
	switch {
	case porcentaje > modelos.UmbralExcedido:
		return modelos.EstadoExcedido
	case porcentaje >= modelos.UmbralAlerta:
		return modelos.EstadoAlerta
	default:
		return modelos.EstadoOK
	}
}
