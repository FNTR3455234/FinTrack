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

func servicioCategorias() (*Categorias, *categoriasFalso, *transaccionesFalso) {
	repoCategorias := nuevoCategoriasFalso()
	repoTransacciones := nuevoTransaccionesFalso()
	return NuevoCategorias(repoCategorias, repoTransacciones), repoCategorias, repoTransacciones
}

func peticionCategoria(nombre, tipo string) modelos.PeticionCategoria {
	return modelos.PeticionCategoria{Nombre: nombre, Tipo: tipo, Color: "#EA580C", Icono: "🛒"}
}

func TestCategoriasCrear_GuardaLaCategoriaConElDueñoDelToken(t *testing.T) {
	servicio, _, _ := servicioCategorias()
	usuarioID := bson.NewObjectID()

	categoria, err := servicio.Crear(context.Background(), usuarioID, peticionCategoria("Supermercado", modelos.TipoGasto))

	require.NoError(t, err)
	assert.Equal(t, usuarioID, categoria.UsuarioID)
	assert.Equal(t, modelos.TipoGasto, categoria.Tipo)
}

func TestCategoriasListar_PuedeFiltrarPorTipo(t *testing.T) {
	servicio, _, _ := servicioCategorias()
	usuarioID := bson.NewObjectID()
	_, err := servicio.Crear(context.Background(), usuarioID, peticionCategoria("Nomina", modelos.TipoIngreso))
	require.NoError(t, err)
	_, err = servicio.Crear(context.Background(), usuarioID, peticionCategoria("Renta", modelos.TipoGasto))
	require.NoError(t, err)

	gastos, err := servicio.Listar(context.Background(), usuarioID, modelos.TipoGasto, false)
	require.NoError(t, err)
	todas, err := servicio.Listar(context.Background(), usuarioID, "", false)
	require.NoError(t, err)

	require.Len(t, gastos, 1)
	assert.Equal(t, "Renta", gastos[0].Nombre)
	assert.Len(t, todas, 2)
}

func TestCategoriasListar_SoloDevuelveLasDelUsuario(t *testing.T) {
	servicio, _, _ := servicioCategorias()
	ana, beto := bson.NewObjectID(), bson.NewObjectID()
	_, err := servicio.Crear(context.Background(), ana, peticionCategoria("De Ana", modelos.TipoGasto))
	require.NoError(t, err)
	_, err = servicio.Crear(context.Background(), beto, peticionCategoria("De Beto", modelos.TipoGasto))
	require.NoError(t, err)

	deAna, err := servicio.Listar(context.Background(), ana, "", false)

	require.NoError(t, err)
	require.Len(t, deAna, 1)
	assert.Equal(t, "De Ana", deAna[0].Nombre)
}

func TestCategoriasPorID_ConLaCategoriaDeOtroUsuarioRespondeNoEncontrada(t *testing.T) {
	servicio, _, _ := servicioCategorias()
	ana, beto := bson.NewObjectID(), bson.NewObjectID()
	deAna, err := servicio.Crear(context.Background(), ana, peticionCategoria("De Ana", modelos.TipoGasto))
	require.NoError(t, err)

	_, err = servicio.PorID(context.Background(), beto, deAna.ID)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCategoriaNoEncontrada, dominio.Codigo)
}

func TestCategoriasActualizar_CambiaLosDatosSinTocarElTipo(t *testing.T) {
	servicio, _, _ := servicioCategorias()
	usuarioID := bson.NewObjectID()
	categoria, err := servicio.Crear(context.Background(), usuarioID, peticionCategoria("Super", modelos.TipoGasto))
	require.NoError(t, err)

	nueva := peticionCategoria("Supermercado", modelos.TipoGasto)
	nueva.Color = "#111111"
	actualizada, err := servicio.Actualizar(context.Background(), usuarioID, categoria.ID, nueva)

	require.NoError(t, err)
	assert.Equal(t, "Supermercado", actualizada.Nombre)
	assert.Equal(t, "#111111", actualizada.Color)
}

func TestCategoriasActualizar_DejaCambiarElTipoSiNoTieneMovimientos(t *testing.T) {
	servicio, _, _ := servicioCategorias()
	usuarioID := bson.NewObjectID()
	categoria, err := servicio.Crear(context.Background(), usuarioID, peticionCategoria("Sin uso", modelos.TipoGasto))
	require.NoError(t, err)

	actualizada, err := servicio.Actualizar(context.Background(), usuarioID, categoria.ID,
		peticionCategoria("Sin uso", modelos.TipoIngreso))

	require.NoError(t, err)
	assert.Equal(t, modelos.TipoIngreso, actualizada.Tipo)
}

func TestCategoriasActualizar_NoDejaCambiarElTipoSiYaTieneMovimientos(t *testing.T) {
	// Dejaria gastos colgando de una categoria de ingreso, y las dos consultas
	// relacionales de la fase 5 darian numeros sin sentido.
	servicio, _, repoTransacciones := servicioCategorias()
	usuarioID := bson.NewObjectID()
	categoria, err := servicio.Crear(context.Background(), usuarioID, peticionCategoria("Super", modelos.TipoGasto))
	require.NoError(t, err)

	movimiento := &modelos.Transaccion{UsuarioID: usuarioID, CategoriaID: categoria.ID, Tipo: modelos.TipoGasto}
	require.NoError(t, repoTransacciones.Crear(context.Background(), movimiento))

	_, err = servicio.Actualizar(context.Background(), usuarioID, categoria.ID,
		peticionCategoria("Super", modelos.TipoIngreso))

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCategoriaConTransacciones, dominio.Codigo)
	assert.Equal(t, 409, dominio.HTTP)
}

func TestCategoriasEliminar_SeNiegaSiTieneMovimientos(t *testing.T) {
	servicio, _, repoTransacciones := servicioCategorias()
	usuarioID := bson.NewObjectID()
	categoria, err := servicio.Crear(context.Background(), usuarioID, peticionCategoria("Super", modelos.TipoGasto))
	require.NoError(t, err)

	movimiento := &modelos.Transaccion{UsuarioID: usuarioID, CategoriaID: categoria.ID}
	require.NoError(t, repoTransacciones.Crear(context.Background(), movimiento))

	err = servicio.Eliminar(context.Background(), usuarioID, categoria.ID)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCategoriaConTransacciones, dominio.Codigo)
}

func TestCategoriasEliminar_BorraLaCategoriaSinMovimientos(t *testing.T) {
	servicio, repo, _ := servicioCategorias()
	usuarioID := bson.NewObjectID()
	categoria, err := servicio.Crear(context.Background(), usuarioID, peticionCategoria("Sin uso", modelos.TipoGasto))
	require.NoError(t, err)

	require.NoError(t, servicio.Eliminar(context.Background(), usuarioID, categoria.ID))
	assert.NotContains(t, repo.datos, categoria.ID)
}

func TestCategoriasEliminar_NoDejaBorrarLaDeOtroUsuario(t *testing.T) {
	servicio, repo, _ := servicioCategorias()
	ana, beto := bson.NewObjectID(), bson.NewObjectID()
	deAna, err := servicio.Crear(context.Background(), ana, peticionCategoria("De Ana", modelos.TipoGasto))
	require.NoError(t, err)

	err = servicio.Eliminar(context.Background(), beto, deAna.ID)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCategoriaNoEncontrada, dominio.Codigo)
	assert.Contains(t, repo.datos, deAna.ID, "la categoria de Ana sigue ahi")
}
