package servicios

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
)

func servicioCuentas() (*Cuentas, *cuentasFalso, *transaccionesFalso) {
	repoCuentas := nuevoCuentasFalso()
	repoTransacciones := nuevoTransaccionesFalso()
	return NuevoCuentas(repoCuentas, repoTransacciones), repoCuentas, repoTransacciones
}

func peticionCuenta(nombre string, saldo float64) modelos.PeticionCuenta {
	return modelos.PeticionCuenta{
		Nombre: nombre, Tipo: modelos.CuentaDebito,
		SaldoInicial: &saldo, Color: "#2563EB",
	}
}

func TestCuentasCrear_GuardaLaCuentaConElDueñoDelToken(t *testing.T) {
	servicio, _, _ := servicioCuentas()
	usuarioID := bson.NewObjectID()

	cuenta, err := servicio.Crear(context.Background(), usuarioID, peticionCuenta("BBVA Debito", 18000))

	require.NoError(t, err)
	assert.Equal(t, usuarioID, cuenta.UsuarioID)
	assert.Equal(t, "BBVA Debito", cuenta.Nombre)
	assert.Equal(t, 18000.0, cuenta.SaldoInicial)
	assert.NotEqual(t, bson.NilObjectID, cuenta.ID)
}

func TestCuentasCrear_AceptaSaldoInicialCeroYNegativo(t *testing.T) {
	// Cero es valido en una cuenta nueva y negativo en una tarjeta de credito.
	// Por eso SaldoInicial es puntero en el DTO.
	servicio, _, _ := servicioCuentas()
	usuarioID := bson.NewObjectID()

	enCero, err := servicio.Crear(context.Background(), usuarioID, peticionCuenta("Nueva", 0))
	require.NoError(t, err)
	assert.Equal(t, 0.0, enCero.SaldoInicial)

	deudora, err := servicio.Crear(context.Background(), usuarioID, peticionCuenta("Credito", -4500))
	require.NoError(t, err)
	assert.Equal(t, -4500.0, deudora.SaldoInicial)
}

func TestCuentasListar_SoloDevuelveLasDelUsuario(t *testing.T) {
	servicio, _, _ := servicioCuentas()
	ana, beto := bson.NewObjectID(), bson.NewObjectID()
	_, err := servicio.Crear(context.Background(), ana, peticionCuenta("De Ana", 100))
	require.NoError(t, err)
	_, err = servicio.Crear(context.Background(), beto, peticionCuenta("De Beto", 200))
	require.NoError(t, err)

	deAna, err := servicio.Listar(context.Background(), ana, false)

	require.NoError(t, err)
	require.Len(t, deAna, 1)
	assert.Equal(t, "De Ana", deAna[0].Nombre)
}

func TestCuentasListar_OcultaLasArchivadasSalvoQueSePidan(t *testing.T) {
	servicio, _, _ := servicioCuentas()
	usuarioID := bson.NewObjectID()
	_, err := servicio.Crear(context.Background(), usuarioID, peticionCuenta("Activa", 100))
	require.NoError(t, err)

	archivada := peticionCuenta("Archivada", 0)
	archivada.Archivada = true
	_, err = servicio.Crear(context.Background(), usuarioID, archivada)
	require.NoError(t, err)

	sinArchivadas, err := servicio.Listar(context.Background(), usuarioID, false)
	require.NoError(t, err)
	conArchivadas, err := servicio.Listar(context.Background(), usuarioID, true)
	require.NoError(t, err)

	assert.Len(t, sinArchivadas, 1)
	assert.Len(t, conArchivadas, 2)
}

func TestCuentasPorID_ConLaCuentaDeOtroUsuarioRespondeNoEncontrada(t *testing.T) {
	// No responde 403: decir "existe pero no es tuya" ya seria filtrar
	// informacion sobre los datos de otro.
	servicio, _, _ := servicioCuentas()
	ana, beto := bson.NewObjectID(), bson.NewObjectID()
	deAna, err := servicio.Crear(context.Background(), ana, peticionCuenta("De Ana", 100))
	require.NoError(t, err)

	_, err = servicio.PorID(context.Background(), beto, deAna.ID)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCuentaNoEncontrada, dominio.Codigo)
	assert.Equal(t, 404, dominio.HTTP)
}

func TestCuentasActualizar_NoDejaTocarLaCuentaDeOtroUsuario(t *testing.T) {
	servicio, _, _ := servicioCuentas()
	ana, beto := bson.NewObjectID(), bson.NewObjectID()
	deAna, err := servicio.Crear(context.Background(), ana, peticionCuenta("De Ana", 100))
	require.NoError(t, err)

	_, err = servicio.Actualizar(context.Background(), beto, deAna.ID, peticionCuenta("Robada", 999))

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCuentaNoEncontrada, dominio.Codigo)

	// Y la cuenta siguio intacta.
	intacta, err := servicio.PorID(context.Background(), ana, deAna.ID)
	require.NoError(t, err)
	assert.Equal(t, "De Ana", intacta.Nombre)
}

func TestCuentasEliminar_BorraLaCuentaSinMovimientos(t *testing.T) {
	servicio, repo, _ := servicioCuentas()
	usuarioID := bson.NewObjectID()
	cuenta, err := servicio.Crear(context.Background(), usuarioID, peticionCuenta("Vacia", 0))
	require.NoError(t, err)

	err = servicio.Eliminar(context.Background(), usuarioID, cuenta.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.datos, cuenta.ID)
}

func TestCuentasEliminar_SeNiegaSiLaCuentaTieneMovimientos(t *testing.T) {
	// No hay borrado en cascada: perder los movimientos por equivocacion no
	// tiene vuelta atras. El cliente debe archivar en su lugar.
	servicio, _, repoTransacciones := servicioCuentas()
	usuarioID := bson.NewObjectID()
	cuenta, err := servicio.Crear(context.Background(), usuarioID, peticionCuenta("Con movimientos", 0))
	require.NoError(t, err)

	movimiento := &modelos.Transaccion{UsuarioID: usuarioID, CuentaID: cuenta.ID}
	require.NoError(t, repoTransacciones.Crear(context.Background(), movimiento))

	err = servicio.Eliminar(context.Background(), usuarioID, cuenta.ID)

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCuentaConTransacciones, dominio.Codigo)
	assert.Equal(t, 409, dominio.HTTP)
	assert.Contains(t, dominio.Mensaje, "Archivala")
}

func TestCuentasEliminar_NoCuentaLosMovimientosDeOtroUsuario(t *testing.T) {
	// El conteo tambien filtra por usuario: los movimientos de Beto no pueden
	// impedirle a Ana borrar su cuenta.
	servicio, _, repoTransacciones := servicioCuentas()
	ana, beto := bson.NewObjectID(), bson.NewObjectID()
	cuentaDeAna, err := servicio.Crear(context.Background(), ana, peticionCuenta("De Ana", 0))
	require.NoError(t, err)

	deBeto := &modelos.Transaccion{UsuarioID: beto, CuentaID: cuentaDeAna.ID}
	require.NoError(t, repoTransacciones.Crear(context.Background(), deBeto))

	assert.NoError(t, servicio.Eliminar(context.Background(), ana, cuentaDeAna.ID))
}

func TestCuentasEliminar_ConUnIDInexistenteRespondeNoEncontrada(t *testing.T) {
	servicio, _, _ := servicioCuentas()

	err := servicio.Eliminar(context.Background(), bson.NewObjectID(), bson.NewObjectID())

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCuentaNoEncontrada, dominio.Codigo)
}

func TestCuentas_TraduceUnaFallaDeLaBaseAErrorInterno(t *testing.T) {
	servicio, repo, _ := servicioCuentas()
	repo.errorForzado = errors.New("mongo: connection refused")

	_, err := servicio.Crear(context.Background(), bson.NewObjectID(), peticionCuenta("X", 0))

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoErrorInterno, dominio.Codigo)
}
