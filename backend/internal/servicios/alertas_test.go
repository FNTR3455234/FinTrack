package servicios

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

// presupuestar le pone un techo a la categoria de gasto del escenario.
func (e escenario) presupuestar(t *testing.T, limite float64) {
	t.Helper()

	require.NoError(t, e.repoPresupuestos.Crear(context.Background(), &modelos.Presupuesto{
		UsuarioID:   e.usuarioID,
		CategoriaID: e.categoriaGasto,
		MontoLimite: limite,
		Mes:         7,
		Anio:        2026,
	}))
}

// gastar registra un gasto de julio de 2026 y devuelve lo que respondio la API.
func (e escenario) gastar(t *testing.T, monto float64) *modelos.TransaccionCreada {
	t.Helper()

	peticion := e.peticion()
	peticion.Monto = monto

	creada, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)
	require.NoError(t, err)
	return creada
}

func TestAlerta_NoAvisaCuandoLaCategoriaNoTienePresupuesto(t *testing.T) {
	e := nuevoEscenario(t)

	creada := e.gastar(t, 3000)

	assert.Nil(t, creada.Alerta, "sin presupuesto no hay nada contra que comparar")
}

func TestAlerta_NoAvisaMientrasElGastoVaHolgado(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000)

	creada := e.gastar(t, 1000) // 25 %

	assert.Nil(t, creada.Alerta)
}

func TestAlerta_AvisaAlLlegarAlUmbral(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000)

	// Justo el 80 %: el umbral entra, no se queda fuera por un peso.
	creada := e.gastar(t, 3200)

	require.NotNil(t, creada.Alerta)
	assert.Equal(t, modelos.EstadoAlerta, creada.Alerta.Estado)
	assert.Equal(t, 80.0, creada.Alerta.PorcentajeUsado)
	assert.Equal(t, 800.0, creada.Alerta.Disponible)
	assert.Contains(t, creada.Alerta.Mensaje, "Supermercado")
}

func TestAlerta_AvisaCuandoYaSePaso(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000)

	creada := e.gastar(t, 4118.10)

	require.NotNil(t, creada.Alerta)
	assert.Equal(t, modelos.EstadoExcedido, creada.Alerta.Estado)
	assert.Equal(t, 4118.10, creada.Alerta.Gastado)
	assert.InDelta(t, -118.10, creada.Alerta.Disponible, 0.001, "el disponible se va a negativo")
	assert.Contains(t, creada.Alerta.Mensaje, "118.10", "el mensaje dice por cuanto se paso")
}

func TestAlerta_SumaTodosLosGastosDelMesNoSoloElUltimo(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000)

	e.gastar(t, 2000)
	creada := e.gastar(t, 1500) // acumulado 3500 = 87.5 %

	require.NotNil(t, creada.Alerta)
	assert.Equal(t, 3500.0, creada.Alerta.Gastado)
	assert.Equal(t, 87.5, creada.Alerta.PorcentajeUsado)
}

func TestAlerta_IncluyeElMovimientoQueSeAcabaDeRegistrar(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000)

	// Un solo gasto que ya rebasa: si la alerta se calculara antes de guardar,
	// este total seria 0 y no habria aviso.
	creada := e.gastar(t, 4500)

	require.NotNil(t, creada.Alerta)
	assert.Equal(t, 4500.0, creada.Alerta.Gastado)
}

func TestAlerta_NoAvisaEnUnIngreso(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000)

	peticion := e.peticion()
	peticion.Tipo = modelos.TipoIngreso
	peticion.CategoriaID = e.categoriaIngreso.Hex()
	peticion.Monto = 50000

	creada, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	require.NoError(t, err)
	assert.Nil(t, creada.Alerta, "un ingreso no gasta presupuesto")
}

func TestAlerta_MiraElPresupuestoDelMesDelMovimientoNoElDeHoy(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000) // solo julio de 2026

	peticion := e.peticion()
	peticion.Monto = 3900
	peticion.Fecha = time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	creada, err := e.servicio.Crear(context.Background(), e.usuarioID, peticion)

	require.NoError(t, err)
	assert.Nil(t, creada.Alerta, "agosto no tiene presupuesto propio")
}

func TestAlerta_UnFalloAlRevisarNoTumbaLaTransaccion(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000)
	e.evaluador.errorForzado = errors.New("la base no respondio")

	creada, err := e.servicio.Crear(context.Background(), e.usuarioID, e.peticion())

	// La transaccion ya esta guardada: responder 500 haria creer al usuario que
	// no se registro y que tiene que volver a capturarla.
	require.NoError(t, err)
	require.NotNil(t, creada)
	assert.False(t, creada.ID.IsZero(), "el movimiento si se guardo")
	assert.Nil(t, creada.Alerta)
}

func TestAlerta_NoSeCruzaConElPresupuestoDeOtroUsuario(t *testing.T) {
	e := nuevoEscenario(t)
	e.presupuestar(t, 4000)

	// Alguien mas se pasa de SU presupuesto en la misma categoria... que no es
	// la misma, porque cada usuario tiene las suyas. Lo que se comprueba aqui es
	// que el gasto ajeno no suma en el conteo de nuestro usuario.
	otro := e.repoTransacciones
	require.NoError(t, otro.Crear(context.Background(), &modelos.Transaccion{
		UsuarioID:   bson.NewObjectID(),
		CategoriaID: e.categoriaGasto,
		Tipo:        modelos.TipoGasto,
		Monto:       9000,
		Fecha:       time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
	}))

	creada := e.gastar(t, 1000)

	assert.Nil(t, creada.Alerta, "los 9000 de otro usuario no cuentan aqui")
}
