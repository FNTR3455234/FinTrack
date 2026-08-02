package servicios

import (
	"context"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// --- metas ------------------------------------------------------------------

type metasFalso struct {
	datos map[bson.ObjectID]*modelos.Meta
}

func nuevoMetasFalso() *metasFalso {
	return &metasFalso{datos: map[bson.ObjectID]*modelos.Meta{}}
}

func (r *metasFalso) Crear(_ context.Context, m *modelos.Meta) error {
	m.ID = bson.NewObjectID()
	copia := *m
	r.datos[m.ID] = &copia
	return nil
}

func (r *metasFalso) PorID(_ context.Context, usuarioID, id bson.ObjectID) (*modelos.Meta, error) {
	m, existe := r.datos[id]
	if !existe || m.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	copia := *m
	return &copia, nil
}

func (r *metasFalso) Actualizar(_ context.Context, usuarioID, id bson.ObjectID, m modelos.Meta) (*modelos.Meta, error) {
	guardada, existe := r.datos[id]
	if !existe || guardada.UsuarioID != usuarioID {
		return nil, repositorios.ErrNoEncontrado
	}
	guardada.Nombre = m.Nombre
	guardada.MontoObjetivo = m.MontoObjetivo
	guardada.FechaLimite = m.FechaLimite
	guardada.Color = m.Color
	guardada.Notas = m.Notas
	guardada.Archivada = m.Archivada
	guardada.ActualizadoEn = m.ActualizadoEn
	copia := *guardada
	return &copia, nil
}

func (r *metasFalso) Eliminar(_ context.Context, usuarioID, id bson.ObjectID) error {
	m, existe := r.datos[id]
	if !existe || m.UsuarioID != usuarioID {
		return repositorios.ErrNoEncontrado
	}
	delete(r.datos, id)
	return nil
}

// --- aportaciones -----------------------------------------------------------

type aportacionesFalso struct {
	datos map[bson.ObjectID]*modelos.Aportacion
}

func nuevoAportacionesFalso() *aportacionesFalso {
	return &aportacionesFalso{datos: map[bson.ObjectID]*modelos.Aportacion{}}
}

func (r *aportacionesFalso) Crear(_ context.Context, a *modelos.Aportacion) error {
	a.ID = bson.NewObjectID()
	copia := *a
	r.datos[a.ID] = &copia
	return nil
}

func (r *aportacionesFalso) DeMeta(_ context.Context, usuarioID, metaID bson.ObjectID) ([]modelos.Aportacion, error) {
	aportaciones := []modelos.Aportacion{}
	for _, a := range r.datos {
		if a.UsuarioID == usuarioID && a.MetaID == metaID {
			aportaciones = append(aportaciones, *a)
		}
	}
	sort.Slice(aportaciones, func(i, j int) bool {
		return aportaciones[i].Fecha.After(aportaciones[j].Fecha)
	})
	return aportaciones, nil
}

// Eliminar filtra por las tres llaves igual que el repositorio de verdad: sin
// eso, la prueba de que no se puede borrar una aportacion de otra meta pasaria
// sin comprobar nada.
func (r *aportacionesFalso) Eliminar(_ context.Context, usuarioID, metaID, id bson.ObjectID) error {
	a, existe := r.datos[id]
	if !existe || a.UsuarioID != usuarioID || a.MetaID != metaID {
		return repositorios.ErrNoEncontrado
	}
	delete(r.datos, id)
	return nil
}

func (r *aportacionesFalso) EliminarDeMeta(_ context.Context, usuarioID, metaID bson.ObjectID) (int64, error) {
	var borradas int64
	for id, a := range r.datos {
		if a.UsuarioID == usuarioID && a.MetaID == metaID {
			delete(r.datos, id)
			borradas++
		}
	}
	return borradas, nil
}

// --- progreso ---------------------------------------------------------------

// progresoFalso imita la agregacion: suma las aportaciones de cada meta. No
// calcula el estado ni el ritmo, igual que la de verdad, porque eso lo pone el
// servicio.
type progresoFalso struct {
	metas        *metasFalso
	aportaciones *aportacionesFalso
}

func (r *progresoFalso) ProgresoMetas(ctx context.Context, usuarioID bson.ObjectID, filtro modelos.FiltroMetas, soloMeta *bson.ObjectID) ([]modelos.ProgresoMeta, error) {
	progreso := []modelos.ProgresoMeta{}

	for id, m := range r.metas.datos {
		if m.UsuarioID != usuarioID {
			continue
		}
		if soloMeta != nil && id != *soloMeta {
			continue
		}
		if soloMeta == nil && m.Archivada && !filtro.IncluirArchivadas {
			continue
		}

		aportaciones, _ := r.aportaciones.DeMeta(ctx, usuarioID, id)
		var ahorrado float64
		for _, a := range aportaciones {
			ahorrado += a.Monto
		}

		fila := modelos.ProgresoMeta{
			MetaID:        id,
			Nombre:        m.Nombre,
			Color:         m.Color,
			MontoObjetivo: m.MontoObjetivo,
			FechaLimite:   m.FechaLimite,
			Notas:         m.Notas,
			Archivada:     m.Archivada,
			Ahorrado:      redondear(ahorrado),
			Restante:      redondear(max(0, m.MontoObjetivo-ahorrado)),
			Porcentaje:    redondear(ahorrado / m.MontoObjetivo * 100),
			Aportaciones:  len(aportaciones),
		}
		if len(aportaciones) > 0 {
			ultima := aportaciones[0].Fecha
			fila.UltimaFecha = &ultima
		}
		progreso = append(progreso, fila)
	}

	sort.Slice(progreso, func(i, j int) bool {
		return progreso[i].FechaLimite.Before(progreso[j].FechaLimite)
	})
	return progreso, nil
}
