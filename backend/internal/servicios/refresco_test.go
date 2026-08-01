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

// --- Refresco ---------------------------------------------------------------

func TestRefrescar_ConUnTokenValidoEntregaUnAccesoNuevo(t *testing.T) {
	auth, _ := servicioDePrueba()
	sesion, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	renovado, err := auth.Refrescar(context.Background(), modelos.PeticionRefresco{
		TokenRefresco: sesion.TokenRefresco,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, renovado.TokenAcceso)
	assert.Equal(t, 15*60, renovado.ExpiraEn)

	// El token nuevo debe servir de verdad y apuntar al mismo usuario.
	usuarioID, err := tokensDePrueba().ValidarAcceso(renovado.TokenAcceso)
	require.NoError(t, err)
	assert.Equal(t, sesion.Usuario.ID, usuarioID)
}

func TestRefrescar_NoAceptaUnTokenDeAcceso(t *testing.T) {
	auth, _ := servicioDePrueba()
	sesion, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	_, err = auth.Refrescar(context.Background(), modelos.PeticionRefresco{
		TokenRefresco: sesion.TokenAcceso,
	})

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTokenInvalido, dominio.Codigo)
}

func TestRefrescar_RechazaUnTokenDeUnUsuarioQueYaNoExiste(t *testing.T) {
	// El token de refresco dura 7 dias: el usuario pudo borrarse en ese tiempo.
	auth, repo := servicioDePrueba()
	sesion, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)
	delete(repo.porID, sesion.Usuario.ID)

	_, err = auth.Refrescar(context.Background(), modelos.PeticionRefresco{
		TokenRefresco: sesion.TokenRefresco,
	})

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTokenInvalido, dominio.Codigo)
}

func TestRefrescar_RechazaUnaCuentaDesactivadaDespuesDeEmitirElToken(t *testing.T) {
	auth, repo := servicioDePrueba()
	sesion, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)
	repo.porID[sesion.Usuario.ID].Activo = false

	_, err = auth.Refrescar(context.Background(), modelos.PeticionRefresco{
		TokenRefresco: sesion.TokenRefresco,
	})

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCuentaDesactivada, dominio.Codigo)
}

func TestRefrescar_RechazaBasura(t *testing.T) {
	auth, _ := servicioDePrueba()

	_, err := auth.Refrescar(context.Background(), modelos.PeticionRefresco{
		TokenRefresco: "esto-no-es-un-token",
	})

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoTokenInvalido, dominio.Codigo)
}

// --- Perfil -----------------------------------------------------------------

func TestPerfil_DevuelveElUsuarioDelToken(t *testing.T) {
	auth, _ := servicioDePrueba()
	sesion, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	usuario, err := auth.Perfil(context.Background(), sesion.Usuario.ID)

	require.NoError(t, err)
	assert.Equal(t, "demo@fintrack.mx", usuario.Email)
}

func TestPerfil_ConUnIDQueNoExisteResponde404(t *testing.T) {
	auth, _ := servicioDePrueba()

	_, err := auth.Perfil(context.Background(), bson.NewObjectID())

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoUsuarioNoEncontrado, dominio.Codigo)
	assert.Equal(t, 404, dominio.HTTP)
}

func TestActualizarPerfil_CambiaNombreYMonedaPeroNoElCorreo(t *testing.T) {
	auth, _ := servicioDePrueba()
	sesion, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	actualizado, err := auth.ActualizarPerfil(context.Background(), sesion.Usuario.ID,
		modelos.PeticionActualizarPerfil{Nombre: "  Nombre Nuevo  ", Moneda: "usd"})

	require.NoError(t, err)
	assert.Equal(t, "Nombre Nuevo", actualizado.Nombre)
	assert.Equal(t, "USD", actualizado.Moneda)
	assert.Equal(t, "demo@fintrack.mx", actualizado.Email, "el correo no se toca")
}

func TestActualizarPerfil_ConUnIDQueNoExisteResponde404(t *testing.T) {
	auth, _ := servicioDePrueba()

	_, err := auth.ActualizarPerfil(context.Background(), bson.NewObjectID(),
		modelos.PeticionActualizarPerfil{Nombre: "X Y", Moneda: "MXN"})

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoUsuarioNoEncontrado, dominio.Codigo)
}
