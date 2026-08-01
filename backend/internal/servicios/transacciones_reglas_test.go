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

func TestTransaccionesActualizar_CambiaLosDatosYConservaCreadoEn(t *testing.T) {
	e := nuevoEscenario(t)
	original, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	peticion := e.peticion()
	peticion.Monto = 1200
	peticion.Descripcion = "Despensa quincenal"

	actualizada, err := e.servicio.Actualizar(context.Background(), e.usuarioID, original.ID, peticion)

	require.NoError(t, err)
	assert.Equal(t, 1200.0, actualizada.Monto)
	assert.Equal(t, "Despensa quincenal", actualizada.Descripcion)
	assert.Equal(t, original.CreadoEn, actualizada.CreadoEn, "creado_en no se toca")
	assert.False(t, actualizada.ActualizadoEn.Before(original.ActualizadoEn))
}

func TestTransaccionesActualizar_TambienValidaLasReferencias(t *testing.T) {
	e := nuevoEscenario(t)
	original, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	peticion := e.peticion()
	peticion.CategoriaID = e.categoriaIngreso.Hex() // tipo gasto contra categoria de ingreso

	_, err = e.servicio.Actualizar(context.Background(), e.usuarioID, original.ID, peticion)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTipoNoCoincide, dominio.Codigo)
}

func TestTransaccionesActualizar_NoDejaTocarLaDeOtroUsuario(t *testing.T) {
	e := nuevoEscenario(t)
	mia, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	// El intruso tiene su propia cuenta y categoria validas, pero la
	// transaccion no es suya.
	intruso := bson.NewObjectID()
	suCuenta := &modelos.Cuenta{UsuarioID: intruso}
	require.NoError(t, e.repoCuentas.Crear(context.Background(), suCuenta))
	suCategoria := &modelos.Categoria{UsuarioID: intruso, Tipo: modelos.TipoGasto}
	require.NoError(t, e.repoCategorias.Crear(context.Background(), suCategoria))

	peticion := e.peticion()
	peticion.CuentaID = suCuenta.ID.Hex()
	peticion.CategoriaID = suCategoria.ID.Hex()

	_, err = e.servicio.Actualizar(context.Background(), intruso, mia.ID, peticion)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTransaccionNoEncontrada, dominio.Codigo)
}

func TestTransaccionesEliminar_BorraLaPropiaYNoLaAjena(t *testing.T) {
	e := nuevoEscenario(t)
	mia, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	err = e.servicio.Eliminar(context.Background(), bson.NewObjectID(), mia.ID)
	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTransaccionNoEncontrada, dominio.Codigo)
	assert.Contains(t, e.repoTransacciones.datos, mia.ID)

	require.NoError(t, e.servicio.Eliminar(context.Background(), e.usuarioID, mia.ID))
	assert.NotContains(t, e.repoTransacciones.datos, mia.ID)
}

func TestRedondear(t *testing.T) {
	casos := map[float64]float64{
		10.994:  10.99,
		10.995:  11.0,
		850.505: 850.51,
		0.1:     0.1,
		1234:    1234,
	}
	for entrada, esperado := range casos {
		assert.Equal(t, esperado, redondear(entrada), "redondear(%v)", entrada)
	}
}

func TestNotasONulo(t *testing.T) {
	assert.Nil(t, notasONulo(""))
	assert.Nil(t, notasONulo("   "))

	notas := notasONulo("  algo  ")
	require.NotNil(t, notas)
	assert.Equal(t, "algo", *notas)
}
