package servicios

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// El reloj de las pruebas esta fijo: sin eso, "faltan 57 dias" seria una
// afirmacion que cambia cada noche.
var hoyDePrueba = dia(2026, time.August, 2)

type escenarioMetas struct {
	servicio     *Metas
	metas        *metasFalso
	aportaciones *aportacionesFalso
	usuarioID    bson.ObjectID
	intrusoID    bson.ObjectID
}

func nuevoEscenarioMetas() escenarioMetas {
	metas := nuevoMetasFalso()
	aportaciones := nuevoAportacionesFalso()
	progreso := &progresoFalso{metas: metas, aportaciones: aportaciones}

	return escenarioMetas{
		servicio:     NuevoMetas(metas, aportaciones, progreso, func() time.Time { return hoyDePrueba }),
		metas:        metas,
		aportaciones: aportaciones,
		usuarioID:    bson.NewObjectID(),
		intrusoID:    bson.NewObjectID(),
	}
}

func (e escenarioMetas) peticion() modelos.PeticionMeta {
	return modelos.PeticionMeta{
		Nombre:        "Fondo de emergencia",
		MontoObjetivo: 30000,
		FechaLimite:   dia(2026, time.October, 31),
		Color:         "#0891B2",
	}
}

func (e escenarioMetas) crear(t *testing.T) *modelos.Meta {
	t.Helper()
	meta, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)
	return meta
}

func TestMetasCrear_GuardaConElDueñoDelTokenYElDiaAnclado(t *testing.T) {
	e := nuevoEscenarioMetas()

	// La fecha llega con hora y con huso, como la mandaria un navegador.
	peticion := e.peticion()
	peticion.FechaLimite = time.Date(2026, time.October, 31, 19, 0, 0, 0, time.FixedZone("CDMX", -6*3600))

	meta, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)
	require.NoError(t, err)

	assert.Equal(t, e.usuarioID, meta.UsuarioID, "el dueño sale del token")
	// El dia que vio el usuario era el 31, y es el que se guarda.
	assert.Equal(t, dia(2026, time.October, 31), meta.FechaLimite)
	assert.Nil(t, meta.Notas, "sin notas se guarda null, no cadena vacia")
}

func TestMetasListar_DevuelveProgresoYResumen(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)

	aportar(t, e, meta.ID, 12000, dia(2026, time.July, 15))
	aportar(t, e, meta.ID, 6500, dia(2026, time.August, 1))

	metas, resumen, err := e.servicio.Listar(context.Background(), e.usuarioID, modelos.FiltroMetas{})
	require.NoError(t, err)
	require.Len(t, metas, 1)

	assert.Equal(t, 18500.0, metas[0].Ahorrado)
	assert.Equal(t, 11500.0, metas[0].Restante)
	assert.InDelta(t, 61.67, metas[0].Porcentaje, 0.01)
	assert.Equal(t, 2, metas[0].Aportaciones)
	assert.Equal(t, modelos.EstadoMetaEnCurso, metas[0].Estado)
	assert.Equal(t, 90, metas[0].DiasRestantes)
	assert.Equal(t, 3833.33, metas[0].RitmoMensual, "11 500 en tres meses")

	assert.Equal(t, 1, resumen.Total)
	assert.Equal(t, 30000.0, resumen.Objetivo)
	assert.Equal(t, 18500.0, resumen.Ahorrado)
}

func TestMetasListar_NoVeLasDeOtroUsuario(t *testing.T) {
	e := nuevoEscenarioMetas()
	e.crear(t)

	metas, resumen, err := e.servicio.Listar(context.Background(), e.intrusoID, modelos.FiltroMetas{})
	require.NoError(t, err)

	assert.Empty(t, metas)
	assert.Zero(t, resumen.Total)
}

func TestMetasListar_LasArchivadasSoloSiSePiden(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)

	archivada := e.peticion()
	archivada.Archivada = true
	_, err := e.servicio.Actualizar(context.Background(), e.usuarioID, meta.ID, archivada)
	require.NoError(t, err)

	visibles, _, err := e.servicio.Listar(context.Background(), e.usuarioID, modelos.FiltroMetas{})
	require.NoError(t, err)
	assert.Empty(t, visibles, "una meta archivada ya no se esta persiguiendo")

	todas, _, err := e.servicio.Listar(context.Background(), e.usuarioID,
		modelos.FiltroMetas{IncluirArchivadas: true})
	require.NoError(t, err)
	assert.Len(t, todas, 1)
}

func TestMetasPorID_TraeElDetalleDeLasAportaciones(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)
	aportar(t, e, meta.ID, 4000, dia(2026, time.March, 15))
	aportar(t, e, meta.ID, 6000, dia(2026, time.May, 15))

	detalle, err := e.servicio.PorID(context.Background(), e.usuarioID, meta.ID)
	require.NoError(t, err)

	assert.Equal(t, 10000.0, detalle.Ahorrado)
	require.Len(t, detalle.Detalle, 2)
	assert.True(t, detalle.Detalle[0].Fecha.After(detalle.Detalle[1].Fecha), "de la mas reciente a la mas vieja")
}

func TestMetasPorID_LaDeOtroUsuarioNoExiste(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)

	_, err := e.servicio.PorID(context.Background(), e.intrusoID, meta.ID)

	dominio, esDominio := fintrackErrores.Como(err)
	require.True(t, esDominio)
	// 404 y no 403: un 403 confirmaria que la meta existe.
	assert.Equal(t, fintrackErrores.CodigoMetaNoEncontrada, dominio.Codigo)
	assert.Equal(t, 404, dominio.HTTP)
}

func TestMetasActualizar_SubirElObjetivoNoBorraLoAhorrado(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)
	aportar(t, e, meta.ID, 15000, dia(2026, time.July, 1))

	masGrande := e.peticion()
	masGrande.MontoObjetivo = 50000
	_, err := e.servicio.Actualizar(context.Background(), e.usuarioID, meta.ID, masGrande)
	require.NoError(t, err)

	detalle, err := e.servicio.PorID(context.Background(), e.usuarioID, meta.ID)
	require.NoError(t, err)

	assert.Equal(t, 15000.0, detalle.Ahorrado, "las aportaciones no se tocan")
	assert.Equal(t, 35000.0, detalle.Restante)
	assert.Equal(t, 30.0, detalle.Porcentaje)
}

func TestMetasEliminar_SeLlevaSusAportaciones(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)
	otra := e.crear(t)
	aportar(t, e, meta.ID, 4000, dia(2026, time.June, 1))
	aportar(t, e, meta.ID, 3000, dia(2026, time.July, 1))
	aportar(t, e, otra.ID, 1000, dia(2026, time.July, 1))

	require.NoError(t, e.servicio.Eliminar(context.Background(), e.usuarioID, meta.ID))

	// Las de la meta borrada se fueron con ella; las de la otra siguen ahi.
	assert.Len(t, e.aportaciones.datos, 1)
	_, err := e.servicio.PorID(context.Background(), e.usuarioID, meta.ID)
	assert.Error(t, err)
}

func TestMetasEliminar_LaDeOtroUsuarioNoSeToca(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)
	aportar(t, e, meta.ID, 4000, dia(2026, time.June, 1))

	err := e.servicio.Eliminar(context.Background(), e.intrusoID, meta.ID)

	assert.Error(t, err)
	assert.Len(t, e.metas.datos, 1, "la meta sigue")
	assert.Len(t, e.aportaciones.datos, 1, "y sus aportaciones tambien")
}

// aportar registra una aportacion y falla la prueba si no se pudo.
func aportar(t *testing.T, e escenarioMetas, metaID bson.ObjectID, monto float64, fecha time.Time) {
	t.Helper()
	_, err := e.servicio.Aportar(context.Background(), e.usuarioID, metaID,
		modelos.PeticionAportacion{Monto: monto, Fecha: fecha})
	require.NoError(t, err)
}
