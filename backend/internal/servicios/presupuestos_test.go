package servicios

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// escenarioPresupuestos deja un usuario con una categoria de gasto y otra de
// ingreso.
type escenarioPresupuestos struct {
	servicio         *Presupuestos
	repo             *presupuestosFalso
	usuarioID        bson.ObjectID
	categoriaGasto   bson.ObjectID
	categoriaIngreso bson.ObjectID
}

func nuevoEscenarioPresupuestos(t *testing.T) escenarioPresupuestos {
	t.Helper()

	repoPresupuestos := nuevoPresupuestosFalso()
	repoCategorias := nuevoCategoriasFalso()
	usuarioID := bson.NewObjectID()

	gasto := &modelos.Categoria{UsuarioID: usuarioID, Nombre: "Supermercado", Tipo: modelos.TipoGasto}
	require.NoError(t, repoCategorias.Crear(context.Background(), gasto))

	ingreso := &modelos.Categoria{UsuarioID: usuarioID, Nombre: "Nomina", Tipo: modelos.TipoIngreso}
	require.NoError(t, repoCategorias.Crear(context.Background(), ingreso))

	return escenarioPresupuestos{
		servicio:         NuevoPresupuestos(repoPresupuestos, repoCategorias),
		repo:             repoPresupuestos,
		usuarioID:        usuarioID,
		categoriaGasto:   gasto.ID,
		categoriaIngreso: ingreso.ID,
	}
}

func (e escenarioPresupuestos) peticion() modelos.PeticionPresupuesto {
	return modelos.PeticionPresupuesto{
		CategoriaID: e.categoriaGasto.Hex(),
		MontoLimite: 4000,
		Mes:         7,
		Anio:        2026,
	}
}

func TestPresupuestosCrear_GuardaElLimiteConElDueñoDelToken(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)

	presupuesto, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())

	require.NoError(t, err)
	assert.Equal(t, e.usuarioID, presupuesto.UsuarioID, "el dueño sale del token, no del cuerpo")
	assert.Equal(t, e.categoriaGasto, presupuesto.CategoriaID)
	assert.Equal(t, 4000.0, presupuesto.MontoLimite)
	assert.Equal(t, modelos.Periodo{Mes: 7, Anio: 2026}, presupuesto.Periodo())
}

func TestPresupuestosCrear_RedondeaElLimiteACentavos(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)

	peticion := e.peticion()
	peticion.MontoLimite = 1234.567

	presupuesto, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	require.NoError(t, err)
	assert.Equal(t, 1234.57, presupuesto.MontoLimite)
}

func TestPresupuestosCrear_RechazaUnaCategoriaDeIngreso(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)

	peticion := e.peticion()
	peticion.CategoriaID = e.categoriaIngreso.Hex()

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTipoNoCoincide, dominio.Codigo)
	assert.Contains(t, dominio.Detalles[0], "Nomina", "el detalle dice cual categoria es")
}

func TestPresupuestosCrear_RechazaUnaCategoriaDeOtroUsuario(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)
	intruso := bson.NewObjectID()

	// La categoria existe, pero es de otro: para el intruso simplemente no esta.
	_, err := e.servicio.Crear(context.Background(), intruso, e.peticion())

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCategoriaNoEncontrada, dominio.Codigo)
}

func TestPresupuestosCrear_RechazaUnIdentificadorMalEscrito(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)

	peticion := e.peticion()
	peticion.CategoriaID = "no-es-un-objectid"

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoIDInvalido, dominio.Codigo)
}

func TestPresupuestosCrear_NoDejaDosPresupuestosParaElMismoMesYCategoria(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	_, err = e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoPresupuestoDuplicado, dominio.Codigo)
	assert.Equal(t, 409, dominio.HTTP)
}

func TestPresupuestosCrear_ElMismoLimiteEnOtroMesSiSePuede(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	agosto := e.peticion()
	agosto.Mes = 8

	_, err = e.servicio.Crear(context.Background(), e.usuarioID, agosto)
	assert.NoError(t, err, "el mismo techo se vuelve a poner cada mes")
}

func TestPresupuestosListar_SoloDevuelveLosDelUsuario(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)
	otro := bson.NewObjectID()

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	ajeno := &modelos.Presupuesto{
		UsuarioID: otro, CategoriaID: bson.NewObjectID(), MontoLimite: 999, Mes: 7, Anio: 2026,
	}
	require.NoError(t, e.repo.Crear(context.Background(), ajeno))

	mios, err := e.servicio.Listar(context.Background(), e.usuarioID, modelos.FiltroPresupuestos{})

	require.NoError(t, err)
	require.Len(t, mios, 1)
	assert.Equal(t, e.usuarioID, mios[0].UsuarioID)
}

func TestPresupuestosListar_FiltraPorPeriodo(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	agosto := e.peticion()
	agosto.Mes = 8
	_, err = e.servicio.Crear(context.Background(), e.usuarioID, agosto)
	require.NoError(t, err)

	periodo := modelos.Periodo{Mes: 8, Anio: 2026}
	deAgosto, err := e.servicio.Listar(context.Background(), e.usuarioID, modelos.FiltroPresupuestos{Periodo: &periodo})

	require.NoError(t, err)
	require.Len(t, deAgosto, 1)
	assert.Equal(t, 8, deAgosto[0].Mes)
}

func TestPresupuestosActualizar_NoAlcanzaElDeOtroUsuario(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)
	intruso := bson.NewObjectID()

	presupuesto, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	// El intruso conoce el id, pero el filtro por usuario_id lo deja fuera. Se
	// responde "no existe" y no "prohibido": un 403 confirmaria que existe.
	_, err = e.servicio.Actualizar(context.Background(), intruso, presupuesto.ID, e.peticion())

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCategoriaNoEncontrada, dominio.Codigo,
		"ni siquiera llega a tocar el presupuesto: su categoria ya no es suya")
}

func TestPresupuestosEliminar_NoAlcanzaElDeOtroUsuario(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)
	intruso := bson.NewObjectID()

	presupuesto, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	err = e.servicio.Eliminar(context.Background(), intruso, presupuesto.ID)
	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoPresupuestoNoEncontrado, dominio.Codigo)

	// Y sigue ahi para su dueño.
	_, err = e.servicio.PorID(context.Background(), e.usuarioID, presupuesto.ID)
	assert.NoError(t, err)
}

func TestPresupuestosEliminar_BorraElPropio(t *testing.T) {
	e := nuevoEscenarioPresupuestos(t)

	presupuesto, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	require.NoError(t, e.servicio.Eliminar(context.Background(), e.usuarioID, presupuesto.ID))

	_, err = e.servicio.PorID(context.Background(), e.usuarioID, presupuesto.ID)
	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoPresupuestoNoEncontrado, dominio.Codigo)
}
