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

func TestAportar_GuardaElDiaAncladoYRedondeaElMonto(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)

	// Fecha con hora y huso, y un monto con mas de dos decimales.
	enCDMX := time.Date(2026, time.July, 31, 19, 0, 0, 0, time.FixedZone("CDMX", -6*3600))
	aportacion, err := e.servicio.Aportar(context.Background(), e.usuarioID, meta.ID,
		modelos.PeticionAportacion{Monto: 1250.999, Fecha: enCDMX, Nota: "  Aguinaldo  "})
	require.NoError(t, err)

	assert.Equal(t, dia(2026, time.July, 31), aportacion.Fecha, "el dia que vio el usuario")
	assert.Equal(t, 1251.0, aportacion.Monto)
	assert.Equal(t, e.usuarioID, aportacion.UsuarioID)
	assert.Equal(t, meta.ID, aportacion.MetaID)
	require.NotNil(t, aportacion.Nota)
	assert.Equal(t, "Aguinaldo", *aportacion.Nota, "la nota se recorta")
}

func TestAportar_NoSePuedeColgarDeUnaMetaQueNoExiste(t *testing.T) {
	e := nuevoEscenarioMetas()

	_, err := e.servicio.Aportar(context.Background(), e.usuarioID, bson.NewObjectID(),
		modelos.PeticionAportacion{Monto: 1000, Fecha: hoyDePrueba})

	dominio, esDominio := fintrackErrores.Como(err)
	require.True(t, esDominio)
	assert.Equal(t, fintrackErrores.CodigoMetaNoEncontrada, dominio.Codigo)
	assert.Empty(t, e.aportaciones.datos, "no queda una aportacion huerfana")
}

func TestAportar_NoSePuedeAportarALaMetaDeOtro(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)

	_, err := e.servicio.Aportar(context.Background(), e.intrusoID, meta.ID,
		modelos.PeticionAportacion{Monto: 1000, Fecha: hoyDePrueba})

	dominio, esDominio := fintrackErrores.Como(err)
	require.True(t, esDominio)
	assert.Equal(t, fintrackErrores.CodigoMetaNoEncontrada, dominio.Codigo)
	assert.Empty(t, e.aportaciones.datos)
}

// Pasarse del objetivo no es un error: juntar mas de lo planeado pasa, y
// bloquearlo obligaria a editar la meta para poder registrar dinero real.
func TestAportar_PermitePasarseDelObjetivo(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)

	aportar(t, e, meta.ID, 35000, hoyDePrueba)

	detalle, err := e.servicio.PorID(context.Background(), e.usuarioID, meta.ID)
	require.NoError(t, err)

	assert.Equal(t, 35000.0, detalle.Ahorrado)
	assert.InDelta(t, 116.67, detalle.Porcentaje, 0.01)
	assert.Zero(t, detalle.Restante, "lo que falta es cero, no un negativo")
	assert.Equal(t, modelos.EstadoMetaCumplida, detalle.Estado)
}

func TestQuitarAportacion_NoBorraLaDeOtraMeta(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)
	otra := e.crear(t)

	aportacion, err := e.servicio.Aportar(context.Background(), e.usuarioID, otra.ID,
		modelos.PeticionAportacion{Monto: 1000, Fecha: hoyDePrueba})
	require.NoError(t, err)

	// Se pide borrarla por la ruta de la meta equivocada.
	err = e.servicio.QuitarAportacion(context.Background(), e.usuarioID, meta.ID, aportacion.ID)

	dominio, esDominio := fintrackErrores.Como(err)
	require.True(t, esDominio)
	assert.Equal(t, fintrackErrores.CodigoAportacionNoEncontrada, dominio.Codigo)
	assert.Len(t, e.aportaciones.datos, 1, "sigue donde estaba")
}

func TestQuitarAportacion_LaBorraYElProgresoBaja(t *testing.T) {
	e := nuevoEscenarioMetas()
	meta := e.crear(t)
	aportar(t, e, meta.ID, 5000, dia(2026, time.June, 1))

	aportacion, err := e.servicio.Aportar(context.Background(), e.usuarioID, meta.ID,
		modelos.PeticionAportacion{Monto: 3000, Fecha: hoyDePrueba})
	require.NoError(t, err)

	require.NoError(t, e.servicio.QuitarAportacion(context.Background(), e.usuarioID, meta.ID, aportacion.ID))

	detalle, err := e.servicio.PorID(context.Background(), e.usuarioID, meta.ID)
	require.NoError(t, err)
	assert.Equal(t, 5000.0, detalle.Ahorrado)
	assert.Equal(t, 1, detalle.Aportaciones)
}
