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

// escenario deja listo un usuario con una cuenta y dos categorias.
type escenario struct {
	servicio          *Transacciones
	repoTransacciones *transaccionesFalso
	repoCuentas       *cuentasFalso
	repoCategorias    *categoriasFalso
	usuarioID         bson.ObjectID
	cuentaID          bson.ObjectID
	categoriaGasto    bson.ObjectID
	categoriaIngreso  bson.ObjectID
}

func nuevoEscenario(t *testing.T) escenario {
	t.Helper()

	repoTransacciones := nuevoTransaccionesFalso()
	repoCuentas := nuevoCuentasFalso()
	repoCategorias := nuevoCategoriasFalso()
	usuarioID := bson.NewObjectID()

	cuenta := &modelos.Cuenta{UsuarioID: usuarioID, Nombre: "BBVA Debito", Tipo: modelos.CuentaDebito}
	require.NoError(t, repoCuentas.Crear(context.Background(), cuenta))

	gasto := &modelos.Categoria{UsuarioID: usuarioID, Nombre: "Supermercado", Tipo: modelos.TipoGasto}
	require.NoError(t, repoCategorias.Crear(context.Background(), gasto))

	ingreso := &modelos.Categoria{UsuarioID: usuarioID, Nombre: "Nomina", Tipo: modelos.TipoIngreso}
	require.NoError(t, repoCategorias.Crear(context.Background(), ingreso))

	return escenario{
		servicio:          NuevoTransacciones(repoTransacciones, repoCuentas, repoCategorias),
		repoTransacciones: repoTransacciones,
		repoCuentas:       repoCuentas,
		repoCategorias:    repoCategorias,
		usuarioID:         usuarioID,
		cuentaID:          cuenta.ID,
		categoriaGasto:    gasto.ID,
		categoriaIngreso:  ingreso.ID,
	}
}

func (e escenario) peticion() modelos.PeticionTransaccion {
	return modelos.PeticionTransaccion{
		CuentaID:    e.cuentaID.Hex(),
		CategoriaID: e.categoriaGasto.Hex(),
		Tipo:        modelos.TipoGasto,
		Monto:       850.50,
		Descripcion: "Despensa de la semana",
		Fecha:       time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
	}
}

func TestTransaccionesCrear_GuardaElMovimientoCompleto(t *testing.T) {
	e := nuevoEscenario(t)

	transaccion, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())

	require.NoError(t, err)
	assert.Equal(t, e.usuarioID, transaccion.UsuarioID)
	assert.Equal(t, e.cuentaID, transaccion.CuentaID)
	assert.Equal(t, 850.50, transaccion.Monto)
	assert.Equal(t, "Despensa de la semana", transaccion.Descripcion)
	assert.False(t, transaccion.CreadoEn.IsZero())
	assert.Equal(t, transaccion.CreadoEn, transaccion.ActualizadoEn)
	assert.Nil(t, transaccion.Notas, "sin notas se guarda null, no cadena vacia")
}

func TestTransaccionesCrear_RedondeaElMontoACentavos(t *testing.T) {
	e := nuevoEscenario(t)
	peticion := e.peticion()
	peticion.Monto = 10.999999999

	transaccion, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	require.NoError(t, err)
	assert.Equal(t, 11.0, transaccion.Monto)
}

func TestTransaccionesCrear_LimpiaLosEspaciosYGuardaLasNotas(t *testing.T) {
	e := nuevoEscenario(t)
	peticion := e.peticion()
	peticion.Descripcion = "  Despensa  "
	peticion.Notas = "  con cupon  "

	transaccion, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	require.NoError(t, err)
	assert.Equal(t, "Despensa", transaccion.Descripcion)
	require.NotNil(t, transaccion.Notas)
	assert.Equal(t, "con cupon", *transaccion.Notas)
}

func TestTransaccionesCrear_ExigeQueElTipoCoincidaConElDeLaCategoria(t *testing.T) {
	// Sin esta regla se podria registrar un "gasto" en la categoria Nomina, y
	// el reporte de gastos por categoria daria numeros sin sentido.
	e := nuevoEscenario(t)
	peticion := e.peticion()
	peticion.CategoriaID = e.categoriaIngreso.Hex() // categoria de ingreso...
	peticion.Tipo = modelos.TipoGasto               // ...con tipo gasto

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTipoNoCoincide, dominio.Codigo)
	assert.Equal(t, 400, dominio.HTTP)
	require.Len(t, dominio.Detalles, 1)
	assert.Contains(t, dominio.Detalles[0], "Nomina")
}

func TestTransaccionesCrear_RechazaUnaCuentaDeOtroUsuario(t *testing.T) {
	// El cuenta_id llega en el cuerpo, o sea que lo elige el cliente. Como la
	// comprobacion filtra por usuario, la cuenta ajena "no existe".
	e := nuevoEscenario(t)
	otroUsuario := bson.NewObjectID()
	cuentaAjena := &modelos.Cuenta{UsuarioID: otroUsuario, Nombre: "De otro"}
	require.NoError(t, e.repoCuentas.Crear(context.Background(), cuentaAjena))

	peticion := e.peticion()
	peticion.CuentaID = cuentaAjena.ID.Hex()

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCuentaNoEncontrada, dominio.Codigo)
}

func TestTransaccionesCrear_RechazaUnaCategoriaDeOtroUsuario(t *testing.T) {
	e := nuevoEscenario(t)
	otroUsuario := bson.NewObjectID()
	categoriaAjena := &modelos.Categoria{UsuarioID: otroUsuario, Nombre: "De otro", Tipo: modelos.TipoGasto}
	require.NoError(t, e.repoCategorias.Crear(context.Background(), categoriaAjena))

	peticion := e.peticion()
	peticion.CategoriaID = categoriaAjena.ID.Hex()

	_, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCategoriaNoEncontrada, dominio.Codigo)
}

func TestTransaccionesCrear_RechazaReferenciasQueNoExisten(t *testing.T) {
	e := nuevoEscenario(t)

	t.Run("cuenta inventada", func(t *testing.T) {
		peticion := e.peticion()
		peticion.CuentaID = bson.NewObjectID().Hex()

		_, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

		dominio, _ := fintrackErrores.Como(err)
		assert.Equal(t, fintrackErrores.CodigoCuentaNoEncontrada, dominio.Codigo)
	})

	t.Run("identificador que no es hexadecimal", func(t *testing.T) {
		peticion := e.peticion()
		peticion.CuentaID = "no-es-un-object-id"

		_, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

		dominio, _ := fintrackErrores.Como(err)
		assert.Equal(t, fintrackErrores.CodigoIDInvalido, dominio.Codigo)
	})
}

func TestTransaccionesListar_PasaElFiltroTalCualAlRepositorio(t *testing.T) {
	e := nuevoEscenario(t)
	desde := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	filtro := modelos.FiltroTransacciones{
		Desde: &desde, Tipo: modelos.TipoGasto, Busqueda: "despensa",
		Pagina: 2, Limite: 50, Orden: modelos.OrdenMontoDesc,
	}

	_, _, err := e.servicio.Listar(context.Background(), e.usuarioID, filtro)

	require.NoError(t, err)
	assert.Equal(t, filtro, e.repoTransacciones.ultimoFiltro)
}

func TestTransaccionesPorID_NoDevuelveLaDeOtroUsuario(t *testing.T) {
	e := nuevoEscenario(t)
	mia, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())
	require.NoError(t, err)

	_, err = e.servicio.PorID(context.Background(), bson.NewObjectID(), mia.ID)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTransaccionNoEncontrada, dominio.Codigo)
}
