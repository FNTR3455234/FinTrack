package servicios

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	fintrackErrores "github.com/FNTR3455234/FinTrack/backend/internal/errores"
	"github.com/FNTR3455234/FinTrack/backend/internal/modelos"
	"github.com/FNTR3455234/FinTrack/backend/internal/repositorios"
)

// repoFalso implementa RepositorioUsuarios en memoria. Permite probar todas las
// reglas del servicio sin levantar MongoDB.
type repoFalso struct {
	porEmail map[string]*modelos.Usuario
	porID    map[bson.ObjectID]*modelos.Usuario
	// errorForzado se usa para simular una caida de la base.
	errorForzado error
}

func nuevoRepoFalso() *repoFalso {
	return &repoFalso{
		porEmail: make(map[string]*modelos.Usuario),
		porID:    make(map[bson.ObjectID]*modelos.Usuario),
	}
}

func (r *repoFalso) Crear(_ context.Context, usuario *modelos.Usuario) error {
	if r.errorForzado != nil {
		return r.errorForzado
	}
	if _, existe := r.porEmail[usuario.Email]; existe {
		return repositorios.ErrDuplicado
	}
	usuario.ID = bson.NewObjectID()
	r.porEmail[usuario.Email] = usuario
	r.porID[usuario.ID] = usuario
	return nil
}

func (r *repoFalso) PorEmail(_ context.Context, email string) (*modelos.Usuario, error) {
	if r.errorForzado != nil {
		return nil, r.errorForzado
	}
	usuario, existe := r.porEmail[email]
	if !existe {
		return nil, repositorios.ErrNoEncontrado
	}
	return usuario, nil
}

func (r *repoFalso) PorID(_ context.Context, id bson.ObjectID) (*modelos.Usuario, error) {
	if r.errorForzado != nil {
		return nil, r.errorForzado
	}
	usuario, existe := r.porID[id]
	if !existe {
		return nil, repositorios.ErrNoEncontrado
	}
	return usuario, nil
}

func (r *repoFalso) Actualizar(_ context.Context, id bson.ObjectID, nombre, moneda string) (*modelos.Usuario, error) {
	usuario, existe := r.porID[id]
	if !existe {
		return nil, repositorios.ErrNoEncontrado
	}
	usuario.Nombre = nombre
	usuario.Moneda = moneda
	return usuario, nil
}

// servicioDePrueba arma el servicio con el repositorio falso.
func servicioDePrueba() (*Auth, *repoFalso) {
	repo := nuevoRepoFalso()
	return NuevoAuth(repo, tokensDePrueba()), repo
}

func registroValido() modelos.PeticionRegistro {
	return modelos.PeticionRegistro{
		Nombre:   "Usuario Demo",
		Email:    "demo@fintrack.mx",
		Password: "Demo1234!",
	}
}

// --- Registro ---------------------------------------------------------------

func TestRegistrar_CreaElUsuarioYDevuelveLaSesion(t *testing.T) {
	auth, repo := servicioDePrueba()

	sesion, err := auth.Registrar(context.Background(), registroValido())

	require.NoError(t, err)
	assert.NotEmpty(t, sesion.TokenAcceso)
	assert.NotEmpty(t, sesion.TokenRefresco)
	assert.Equal(t, 15*60, sesion.ExpiraEn)
	assert.Equal(t, "Usuario Demo", sesion.Usuario.Nombre)
	assert.True(t, sesion.Usuario.Activo)
	assert.Len(t, repo.porEmail, 1)
}

func TestRegistrar_GuardaLaContraseñaComoHashBcryptYNoEnClaro(t *testing.T) {
	auth, repo := servicioDePrueba()

	_, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	guardado := repo.porEmail["demo@fintrack.mx"]
	assert.NotEqual(t, "Demo1234!", guardado.Password)
	assert.NotContains(t, guardado.Password, "Demo1234!")
	// El hash debe poder verificar la contraseña original.
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(guardado.Password), []byte("Demo1234!")))
}

func TestRegistrar_NormalizaElCorreoAMinusculas(t *testing.T) {
	auth, repo := servicioDePrueba()
	peticion := registroValido()
	peticion.Email = "  Demo@FinTrack.MX  "

	_, err := auth.Registrar(context.Background(), peticion)

	require.NoError(t, err)
	assert.Contains(t, repo.porEmail, "demo@fintrack.mx")
}

func TestRegistrar_UsaMXNCuandoNoSeIndicaMoneda(t *testing.T) {
	auth, _ := servicioDePrueba()

	sesion, err := auth.Registrar(context.Background(), registroValido())

	require.NoError(t, err)
	assert.Equal(t, "MXN", sesion.Usuario.Moneda)
}

func TestRegistrar_RespetaLaMonedaIndicadaYLaPasaAMayusculas(t *testing.T) {
	auth, _ := servicioDePrueba()
	peticion := registroValido()
	peticion.Moneda = "usd"

	sesion, err := auth.Registrar(context.Background(), peticion)

	require.NoError(t, err)
	assert.Equal(t, "USD", sesion.Usuario.Moneda)
}

func TestRegistrar_RechazaUnCorreoYaRegistrado(t *testing.T) {
	auth, _ := servicioDePrueba()
	_, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	_, err = auth.Registrar(context.Background(), registroValido())

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoEmailYaRegistrado, dominio.Codigo)
	assert.Equal(t, 409, dominio.HTTP)
}

func TestRegistrar_TraduceUnaFallaDeLaBaseAErrorInterno(t *testing.T) {
	auth, repo := servicioDePrueba()
	repo.errorForzado = errors.New("mongo: connection refused")

	_, err := auth.Registrar(context.Background(), registroValido())

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoErrorInterno, dominio.Codigo)
	assert.Equal(t, 500, dominio.HTTP)
}

// --- Inicio de sesion -------------------------------------------------------

func TestIniciarSesion_ConCredencialesCorrectasDevuelveLosDosTokens(t *testing.T) {
	auth, _ := servicioDePrueba()
	_, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	sesion, err := auth.IniciarSesion(context.Background(), modelos.PeticionLogin{
		Email:    "demo@fintrack.mx",
		Password: "Demo1234!",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, sesion.TokenAcceso)
	assert.NotEmpty(t, sesion.TokenRefresco)
	assert.Equal(t, "demo@fintrack.mx", sesion.Usuario.Email)
}

func TestIniciarSesion_DaElMismoErrorSiElCorreoNoExisteOLaClaveEsIncorrecta(t *testing.T) {
	// Si los errores fueran distintos, cualquiera podria averiguar que correos
	// tienen cuenta probando el login.
	auth, _ := servicioDePrueba()
	_, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	_, errCorreoInexistente := auth.IniciarSesion(context.Background(), modelos.PeticionLogin{
		Email: "nadie@fintrack.mx", Password: "Demo1234!",
	})
	_, errClaveMala := auth.IniciarSesion(context.Background(), modelos.PeticionLogin{
		Email: "demo@fintrack.mx", Password: "otra-cosa",
	})

	unoDominio, ok := fintrackErrores.Como(errCorreoInexistente)
	require.True(t, ok)
	otroDominio, ok := fintrackErrores.Como(errClaveMala)
	require.True(t, ok)

	assert.Equal(t, fintrackErrores.CodigoCredencialesInvalidas, unoDominio.Codigo)
	assert.Equal(t, unoDominio.Codigo, otroDominio.Codigo)
	assert.Equal(t, unoDominio.Mensaje, otroDominio.Mensaje)
	assert.Equal(t, 401, unoDominio.HTTP)
}

func TestIniciarSesion_AceptaElCorreoEnMayusculas(t *testing.T) {
	auth, _ := servicioDePrueba()
	_, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)

	_, err = auth.IniciarSesion(context.Background(), modelos.PeticionLogin{
		Email: "DEMO@FINTRACK.MX", Password: "Demo1234!",
	})

	assert.NoError(t, err)
}

func TestIniciarSesion_RechazaUnaCuentaDesactivada(t *testing.T) {
	auth, repo := servicioDePrueba()
	_, err := auth.Registrar(context.Background(), registroValido())
	require.NoError(t, err)
	repo.porEmail["demo@fintrack.mx"].Activo = false

	_, err = auth.IniciarSesion(context.Background(), modelos.PeticionLogin{
		Email: "demo@fintrack.mx", Password: "Demo1234!",
	})

	dominio, ok := fintrackErrores.Como(err)
	require.True(t, ok)
	assert.Equal(t, fintrackErrores.CodigoCuentaDesactivada, dominio.Codigo)
	assert.Equal(t, 403, dominio.HTTP)
}
